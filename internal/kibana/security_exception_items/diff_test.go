// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package securityexceptionitems

import (
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// makeItem builds a minimal BulkItemModel for diffItems tests. Only the fields
// examined by diffItems (ItemID, ID, Version) need to be populated.
func makeItem(itemID, id, version string) BulkItemModel {
	m := BulkItemModel{}
	m.ItemID = types.StringValue(itemID)
	if id != "" {
		m.ID = types.StringValue(id)
	} else {
		m.ID = types.StringNull()
	}
	if version != "" {
		m.Version = types.StringValue(version)
	} else {
		m.Version = types.StringNull()
	}
	return m
}

func itemIDs(items []BulkItemModel) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ItemID.ValueString()
	}
	sort.Strings(ids)
	return ids
}

func TestDiffItems_EmptyBoth(t *testing.T) {
	toCreate, toUpdate, toDelete := diffItems(nil, nil)
	if len(toCreate)+len(toUpdate)+len(toDelete) != 0 {
		t.Errorf("expected all empty, got create=%v update=%v delete=%v", toCreate, toUpdate, toDelete)
	}
}

func TestDiffItems_AllCreate(t *testing.T) {
	plan := []BulkItemModel{
		makeItem("item-a", "", ""),
		makeItem("item-b", "", ""),
	}
	toCreate, toUpdate, toDelete := diffItems(plan, nil)

	if len(toCreate) != 2 {
		t.Errorf("expected 2 creates, got %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Errorf("expected 0 updates, got %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(toDelete))
	}
}

func TestDiffItems_AllDelete(t *testing.T) {
	state := []BulkItemModel{
		makeItem("item-a", "uuid-a", "v1"),
		makeItem("item-b", "uuid-b", "v2"),
	}
	toCreate, toUpdate, toDelete := diffItems(nil, state)

	if len(toCreate) != 0 {
		t.Errorf("expected 0 creates, got %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Errorf("expected 0 updates, got %d", len(toUpdate))
	}
	if len(toDelete) != 2 {
		t.Errorf("expected 2 deletes, got %d: %v", len(toDelete), toDelete)
	}
}

func TestDiffItems_AllUpdate(t *testing.T) {
	plan := []BulkItemModel{
		makeItem("item-a", "", ""),
		makeItem("item-b", "", ""),
	}
	state := []BulkItemModel{
		makeItem("item-a", "uuid-a", "v1"),
		makeItem("item-b", "uuid-b", "v2"),
	}
	toCreate, toUpdate, toDelete := diffItems(plan, state)

	if len(toCreate) != 0 {
		t.Errorf("expected 0 creates, got %d", len(toCreate))
	}
	if len(toUpdate) != 2 {
		t.Errorf("expected 2 updates, got %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(toDelete))
	}
}

func TestDiffItems_Mixed(t *testing.T) {
	// Plan: A (existing), C (new). State: A (existing), B (removed).
	plan := []BulkItemModel{
		makeItem("item-a", "", ""),
		makeItem("item-c", "", ""),
	}
	state := []BulkItemModel{
		makeItem("item-a", "uuid-a", "v1"),
		makeItem("item-b", "uuid-b", "v2"),
	}
	toCreate, toUpdate, toDelete := diffItems(plan, state)

	createIDs := itemIDs(toCreate)
	if len(createIDs) != 1 || createIDs[0] != "item-c" {
		t.Errorf("expected create=[item-c], got %v", createIDs)
	}

	updateIDs := itemIDs(toUpdate)
	if len(updateIDs) != 1 || updateIDs[0] != "item-a" {
		t.Errorf("expected update=[item-a], got %v", updateIDs)
	}

	if len(toDelete) != 1 || toDelete[0] != "uuid-b" {
		t.Errorf("expected delete=[uuid-b], got %v", toDelete)
	}
}

func TestDiffItems_UpdateCarriesComputedFields(t *testing.T) {
	// The plan item has no computed fields (null ID/Version). The state item has
	// real values. diffItems must copy them onto the update item so the bulk update
	// request carries the correct ID and optimistic concurrency version.
	plan := []BulkItemModel{makeItem("item-a", "", "")}
	state := []BulkItemModel{makeItem("item-a", "uuid-a", "WzEwMCwxXQ==")}

	_, toUpdate, _ := diffItems(plan, state)

	if len(toUpdate) != 1 {
		t.Fatalf("expected 1 update, got %d", len(toUpdate))
	}
	u := toUpdate[0]
	if u.ID.ValueString() != "uuid-a" {
		t.Errorf("ID: got %q, want %q", u.ID.ValueString(), "uuid-a")
	}
	if u.Version.ValueString() != "WzEwMCwxXQ==" {
		t.Errorf("Version: got %q, want %q", u.Version.ValueString(), "WzEwMCwxXQ==")
	}
}

func TestDiffItems_DeleteUsesKibanaID(t *testing.T) {
	// Bulk delete sends the Kibana document UUID, not the user-facing item_id.
	state := []BulkItemModel{makeItem("item-a", "uuid-a", "v1")}
	_, _, toDelete := diffItems(nil, state)

	if len(toDelete) != 1 || toDelete[0] != "uuid-a" {
		t.Errorf("expected delete=[uuid-a], got %v", toDelete)
	}
}

func TestDiffItems_SkipsStateItemWithoutKibanaID(t *testing.T) {
	// A state item without a Kibana ID (empty string) should not generate a
	// delete entry, since we have nothing to send to the bulk delete endpoint.
	state := []BulkItemModel{makeItem("item-a", "", "")}
	_, _, toDelete := diffItems(nil, state)

	if len(toDelete) != 0 {
		t.Errorf("expected 0 deletes, got %v", toDelete)
	}
}
