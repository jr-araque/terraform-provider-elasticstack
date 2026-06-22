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

package securityexceptionitem

import (
	"context"
	"fmt"

	shared "github.com/elastic/terraform-provider-elasticstack/internal/kibana/securityexceptionshared"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Package-level aliases for shared constants, kept for backward compatibility
// within this package (used by validate logic below).
const (
	entryTypeMatch    = shared.EntryTypeMatch
	entryTypeWildcard = shared.EntryTypeWildcard
	entryTypeMatchAny = shared.EntryTypeMatchAny
	entryTypeList     = shared.EntryTypeList
	entryTypeExists   = shared.EntryTypeExists
	entryTypeNested   = shared.EntryTypeNested
	attrType          = shared.AttrType
	attrField         = shared.AttrField
	attrOperator      = shared.AttrOperator
	attrValue         = shared.AttrValue
	attrValues        = shared.AttrValues
	attrEntries       = shared.AttrEntries
)

// ValidateConfig validates the configuration for an exception item resource.
func (r *ExceptionItemResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ExceptionItemModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !typeutils.IsKnown(data.Entries) {
		return
	}

	var entries []EntryModel
	resp.Diagnostics.Append(data.Entries.ElementsAs(ctx, &entries, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, entry := range entries {
		validateEntry(ctx, entry, i, &resp.Diagnostics, "entries")
	}
}

// validateEntry validates a single entry based on its type
func validateEntry(ctx context.Context, entry EntryModel, index int, diags *diag.Diagnostics, path string) {
	if !typeutils.IsKnown(entry.Type) {
		return
	}

	entryType := entry.Type.ValueString()
	entryPath := fmt.Sprintf("%s[%d]", path, index)

	switch entryType {
	case entryTypeMatch, entryTypeWildcard:
		if entry.Value.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'value' to be set at %s.", entryType, entryPath),
			)
		}
		if entry.Operator.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'operator' to be set at %s.", entryType, entryPath),
			)
		}

	case entryTypeMatchAny:
		if entry.Values.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'values' to be set at %s.", entryTypeMatchAny, entryPath),
			)
		}
		if entry.Operator.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'operator' to be set at %s.", entryTypeMatchAny, entryPath),
			)
		}

	case entryTypeList:
		if entry.List.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'list' object to be set at %s.", entryTypeList, entryPath),
			)
		} else if !entry.List.IsUnknown() {
			var listModel EntryListModel
			d := entry.List.As(ctx, &listModel, basetypes.ObjectAsOptions{})
			if d.HasError() {
				diags.Append(d...)
			} else {
				if listModel.ID.IsNull() {
					diags.AddError(
						"Missing Required Field",
						fmt.Sprintf("Entry type '%s' requires 'list.id' to be set at %s.", entryTypeList, entryPath),
					)
				}
				if listModel.Type.IsNull() {
					diags.AddError(
						"Missing Required Field",
						fmt.Sprintf("Entry type '%s' requires 'list.type' to be set at %s.", entryTypeList, entryPath),
					)
				}
			}
		}
		if entry.Operator.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'operator' to be set at %s.", entryTypeList, entryPath),
			)
		}

	case entryTypeExists:
		if entry.Operator.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'operator' to be set at %s.", entryTypeExists, entryPath),
			)
		}

	case entryTypeNested:
		if entry.Entries.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Entry type '%s' requires 'entries' to be set at %s.", entryTypeNested, entryPath),
			)
			return
		}

		if entry.Entries.IsUnknown() {
			return
		}

		if typeutils.IsKnown(entry.Operator) {
			diags.AddWarning(
				"Ignored Field",
				fmt.Sprintf("Entry type '%s' does not support 'operator'. This field will be ignored at %s.", entryTypeNested, entryPath),
			)
		}

		var nestedEntries []NestedEntryModel
		d := entry.Entries.ElementsAs(ctx, &nestedEntries, false)
		if d.HasError() {
			diags.Append(d...)
			return
		}

		for j, nestedEntry := range nestedEntries {
			validateNestedEntry(ctx, nestedEntry, j, diags, fmt.Sprintf("%s.entries", entryPath))
		}
	}
}

// validateNestedEntry validates a nested entry within a "nested" type entry
func validateNestedEntry(ctx context.Context, entry NestedEntryModel, index int, diags *diag.Diagnostics, path string) {
	_ = ctx
	if !typeutils.IsKnown(entry.Type) {
		return
	}

	entryType := entry.Type.ValueString()
	entryPath := fmt.Sprintf("%s[%d]", path, index)

	switch entryType {
	case entryTypeMatch:
		if entry.Value.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Nested entry type '%s' requires 'value' to be set at %s.", entryTypeMatch, entryPath),
			)
		}

	case entryTypeMatchAny:
		if entry.Values.IsNull() {
			diags.AddError(
				"Missing Required Field",
				fmt.Sprintf("Nested entry type '%s' requires 'values' to be set at %s.", entryTypeMatchAny, entryPath),
			)
		}

	case entryTypeExists:
		// Only field and operator required; handled by schema

	default:
		diags.AddError(
			"Invalid Entry Type",
			fmt.Sprintf("Nested entry at %s has invalid type '%s'. Only 'match', 'match_any', and 'exists' are allowed for nested entries.", entryPath, entryType),
		)
	}

	if entry.Operator.IsNull() {
		diags.AddError(
			"Missing Required Field",
			fmt.Sprintf("Nested entry requires 'operator' to be set at %s.", entryPath),
		)
	}
}
