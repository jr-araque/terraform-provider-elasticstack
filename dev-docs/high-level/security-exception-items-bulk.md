# elasticstack_kibana_security_exception_items — bulk exception list items resource

## Why this resource exists

The singular `elasticstack_kibana_security_exception_item` manages one Kibana exception
list item per Terraform resource block. A list with hundreds of items requires hundreds of
resource blocks, state entries, and individual API calls per apply. This resource solves
that by treating the entire item set of a given exception list as a single managed unit,
using Kibana's bulk endpoints for efficient batch operations.

Related issue: #266239 — Kibana bulk CRUD endpoints for exception list items.

---

## Design: aggregate resource pattern

One Terraform resource owns **all items** in a Kibana exception list. The provider computes
the diff between plan and state internally (keyed on `item_id`), then issues bulk API calls
to reconcile. Terraform's built-in state-tracking is not used at the item level.

```hcl
resource "elasticstack_kibana_security_exception_items" "example" {
  list_id        = "my-exception-list"
  namespace_type = "single"   # or "agnostic"

  items {
    item_id     = "allow-svchost"   # required — stable diff key
    type        = "simple"
    name        = "Allow trusted process"
    description = "Allowlist for known-good process"
    entries = [{ type = "match", field = "process.name", operator = "included", value = "svchost.exe" }]
  }
}
```

Key design decisions:

- `item_id` is **Required** (not Optional+Computed). If computed, Terraform uses position-based
  `UseStateForUnknown`, which assigns wrong UUIDs when items are reordered or removed from the
  middle of the list, orphaning items in Kibana on every apply.
- The resource supports up to **10,000 items** (Kibana's exception list capacity).
- All bulk API calls are internally chunked into **1,000-item batches** (Kibana's bulk limit).

---

## CRUD lifecycle

| Lifecycle | Kibana endpoint | Behavior |
|---|---|---|
| **Create** | `POST /api/exception_lists/items/_bulk` | Chunks items into 1,000-item batches sequentially |
| **Read** | `GET /api/exception_lists/items/_find` | Paginated (1,000/page); prior-state order preserved; new items sorted by `item_id` |
| **Update** | 3-way diff by `item_id` → `_bulk` / `_bulk_update` / `_bulk_delete` | All three phases run per apply; transport error in any phase aborts remaining phases |
| **Delete** | `POST /api/exception_lists/items/_bulk_delete` | Chunks all state item IDs |

### Update diff logic (`diffItems`)

Items are matched by `item_id`:

- In plan but not in state → **bulk create**
- In both → **bulk update** (state's `id` and `_version` copied onto the plan item for optimistic concurrency)
- In state but not in plan → **bulk delete** (uses the Kibana document UUID, not `item_id`)

Transport errors (HTTP/network failure) are **fatal** — abort subsequent bulk phases.
Per-item errors inside a successful HTTP response are **accumulated** — report all failures in
one apply output without aborting other items.

After every create or update, the envelope (`entitycore.KibanaResource`) performs a mandatory
**read-after-write** that fetches live Kibana state. `create.go` and `update.go` do not
accumulate result items; they rely on this authoritative read.

---

## Read: deleted-list detection

`read.go` first calls `GetExceptionList` explicitly. A nil return (404) means the parent
exception list was deleted out-of-band, and the resource is marked as not-found so Terraform
will plan a re-create. An empty `_find` result alone is ambiguous (could mean 0 items or
deleted list) — the explicit GET disambiguates.

### Read ordering

Items from the prior state are emitted first, in their stored order (preserving the user's
declared order across refreshes). Items not in prior state (e.g., first import or out-of-band
additions) are sorted by `item_id` before appending, ensuring deterministic state across
repeated Reads.

---

## Import

```sh
# Single (default)
terraform import elasticstack_kibana_security_exception_items.example default/my-list

# Agnostic
terraform import elasticstack_kibana_security_exception_items.example default/my-list/agnostic
```

`namespace_type` is detected via **suffix** (`/agnostic` or `/single`) rather than a positional
split (`strings.SplitN(id, "/", 3)`), because `list_id` may itself contain forward slashes.
"agnostic" and "single" are the only valid values and cannot appear as list_id segments.

The canonical 2-segment composite ID (`<space_id>/<list_id>`) is written explicitly to the `id`
attribute in state. `ImportStatePassthroughID` must not be used here because
`CompositeIDFromStr` uses `SplitN(id, "/", 2)` and would fold any trailing segment (e.g.,
`/agnostic`) into `listID`.

---

## Ownership constraint

> **Do not use this resource alongside `elasticstack_kibana_security_exception_item`
> (singular) for the same `list_id`.**

The bulk resource's Read fetches ALL items from the list. On the next apply it will delete any
items it does not recognise from its own state, including items managed by the singular
resource. There is no programmatic prevention — it is a configuration constraint documented in
the schema description.

---

## Shared package: `securityexceptionshared`

`internal/kibana/securityexceptionshared/` was extracted so both the singular and plural
resources share:

- Entry and comment struct types (`EntryModel`, `NestedEntryModel`, `CommentModel`)
- `ConvertEntriesToAPI` / `ConvertEntriesFromAPI` conversion helpers
- `EntriesSchema()` / `CommentsSchema()` schema helpers
- Attribute type maps (`GetEntryAttrTypes`, `GetCommentAttrTypes`)

The singular resource uses type aliases (`type EntryModel = shared.EntryModel`) for zero-impact
refactoring: no callers were broken.

---

## API client layer

The Kibana OAS spec does not yet include the bulk endpoints. A raw HTTP layer at
`internal/clients/kibanaoapi/exception_items_bulk.go` implements them directly against
`client.HTTP`. This `*http.Client` uses the provider's custom `Transport.RoundTrip`, which
injects Basic auth / API key / Bearer on every request automatically — no extra auth wiring
needed for raw HTTP calls.

URL construction: `strings.TrimSuffix(client.URL, "/") + path` prevents double-slash URLs
when the user configures a trailing slash in their endpoint setting.

`FindExceptionListItemsAllPages` uses the existing generated `FindExceptionListItemsWithResponse`
endpoint with `kibanautil.SpaceAwarePathRequestEditor` to inject the `/s/{spaceId}` prefix.

**When Kibana ships these endpoints in the OAS spec:** add their paths to `spaceIdPaths` in
`generated/kbapi/transform_schema.go`, run `make -C generated/kbapi all`, and replace the raw
HTTP calls in `exception_items_bulk.go` with the generated client.

---

## `refresh=wait_for`

All mutating calls pass `refresh=wait_for` via `kibanautil.WithRefreshWaitFor`. The original
POC used `refresh=false` (return immediately, no visibility guarantee), which races with the
envelope's mandatory read-after-write. `wait_for` blocks until the write is visible in the
next index refresh cycle without forcing a synchronous flush — correct for writes followed by
an immediate read.

---

## Bugs found and fixed during review

| Commit | Bug | Fix |
|---|---|---|
| `9693f059` | Deleted-list detection was dead code: `if _, d := planItems(...); d == nil` — a `diag.Diagnostics` empty slice is never nil in Go | Call `GetExceptionList` first; nil return = 404 = found=false |
| `bafb78f5` | `update.go` accumulated only created+updated items in `resultItems`, omitting unchanged items; also early-returned on first per-item error | Remove `resultItems` entirely; return plan model and rely on read-after-write |
| `a8fc115f` | `item_id` was Optional+Computed; Terraform position-based `UseStateForUnknown` assigned wrong UUIDs when items were reordered, orphaning items on every update | Make `item_id` Required |
| `2ed139e0` | `refresh=false` returned before writes were visible to search, racing with read-after-write | Change to `refresh=wait_for`; rename `WithRefreshFalse` → `WithRefreshWaitFor` |
| `de6bf8fc` | Import did not set `namespace_type` | First attempt (later replaced by a correct implementation) |
| `99aee5fe` | No warning to users about ownership conflict with the singular resource | Document constraint in schema `MarkdownDescription` |
| `bd27c5d5` | `ImportStatePassthroughID` stored the full 3-segment raw ID; `CompositeIDFromStr`'s `SplitN(id,"/",2)` folded `/agnostic` into `listID` | Write only the canonical 2-segment ID to the `id` attribute manually |
| `f543a988` | Transport errors fell through to subsequent bulk phases; `create.go` collected dead `createdItems` (overwritten by read-after-write anyway) | Abort on transport error; remove result accumulation from create and update |
| `0db2414e` | Import positional `SplitN(id,"/",3)` broke when `list_id` contained slashes; `client.URL + path` doubled slash; new items iterated over map → non-deterministic Read order | `strings.HasSuffix` detection; `strings.TrimSuffix(client.URL,"/") + path`; `sort.Strings` before appending |

---

## Tests

### Unit tests (no external deps)

```sh
go test ./internal/kibana/security_exception_items/ -run 'TestDiffItems|TestParseImportID' -v
```

**`diff_test.go`** covers `diffItems`:
- All create (empty state)
- All delete (empty plan)
- All update (matching items)
- Mixed create/update/delete
- Computed fields (ID, Version) copied from state onto update items
- Delete sends Kibana UUID, not `item_id`
- State items without a Kibana UUID are skipped from delete

**`resource_test.go`** covers `parseImportID`:
- No suffix → empty nsType, full string as ID
- `/agnostic` suffix → nsType="agnostic", remainder as ID
- `/single` suffix → nsType="single", remainder as ID
- `list_id` with forward slashes + suffix → correct parse
- Non-default space ID

### Acceptance tests (requires Elastic Stack)

See [testing.md](./testing.md) for environment setup.

```sh
go test ./internal/kibana/security_exception_items/ -v -run TestAccResourceExceptionItems -timeout 30m
```

`acc_test.go` covers: create (2 items) → update-add (3 items + rename) → update-remove (2 items) → import.

**Missing acceptance test coverage** (known gaps):
- Import of an `agnostic` list
- Import with `list_id` containing forward slashes
- List deleted out-of-band between applies (Read should return not-found)
- Partial bulk create/update failure
- More than 1,000 items (chunking path)
- `expire_time` round-trip with a non-UTC offset (potential permadiff)

---

## Known limitations

- **`expire_time` permadiff risk**: Write serializes as `2006-01-02T15:04:05.000Z`; read uses
  `time.RFC3339`. Non-UTC offsets in user config may produce a permadiff. Needs an acceptance
  test to confirm.
- **`os_types`/`tags` drift masking**: When the API returns an empty value, the prior state value
  is preserved to avoid empty/null churn. Out-of-band removals of `os_types` and `tags` are
  silently masked until the next apply that explicitly sets them empty.
- **Bulk endpoints not in generated client**: Swap raw HTTP in `exception_items_bulk.go` for the
  generated client once Kibana ships the OAS spec paths. Also add them to `transform_schema.go`.
