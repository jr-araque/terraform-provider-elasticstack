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

package securityexceptionshared

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ConvertEntriesToAPI converts a Terraform entries list to an API entry array.
func ConvertEntriesToAPI(ctx context.Context, entries types.List) (kbapi.SecurityExceptionsAPIExceptionListItemEntryArray, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !typeutils.IsKnown(entries) {
		return nil, diags
	}

	entryModels := typeutils.ListTypeAs[EntryModel](ctx, entries, path.Empty(), &diags)
	if diags.HasError() {
		return nil, diags
	}

	apiEntries := make(kbapi.SecurityExceptionsAPIExceptionListItemEntryArray, 0, len(entryModels))
	for _, entry := range entryModels {
		apiEntry, d := ConvertEntryToAPI(ctx, entry)
		diags.Append(d...)
		if d.HasError() {
			continue
		}
		apiEntries = append(apiEntries, apiEntry)
	}

	return apiEntries, diags
}

// ConvertEntryToAPI converts a single Terraform entry model to an API entry.
func ConvertEntryToAPI(ctx context.Context, entry EntryModel) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	entryType := entry.Type.ValueString()
	operator := kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator(entry.Operator.ValueString())
	field := entry.Field.ValueString()

	switch entryType {
	case EntryTypeMatch:
		return convertMatchEntryToAPI(entry, field, operator)
	case EntryTypeMatchAny:
		return convertMatchAnyEntryToAPI(ctx, entry, field, operator)
	case EntryTypeList:
		return convertListEntryToAPI(ctx, entry, field, operator)
	case EntryTypeExists:
		return convertExistsEntryToAPI(field, operator)
	case EntryTypeWildcard:
		return convertWildcardEntryToAPI(entry, field, operator)
	case EntryTypeNested:
		return convertNestedEntryArrayToAPI(ctx, entry, field)
	default:
		diags.AddError("Invalid entry type", fmt.Sprintf("Unknown entry type: %s", entryType))
		return result, diags
	}
}

func convertMatchEntryToAPI(
	entry EntryModel,
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	if !typeutils.IsKnown(entry.Value) || entry.Value.ValueString() == "" {
		diags.AddError("Invalid Configuration", "Attribute 'value' is required when type is 'match'")
		return result, diags
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryMatch{
		Type:     EntryTypeMatch,
		Field:    field,
		Operator: operator,
		Value:    entry.Value.ValueString(),
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryMatch(apiEntry); err != nil {
		diags.AddError("Failed to create match entry", err.Error())
	}
	return result, diags
}

func convertMatchAnyEntryToAPI(
	ctx context.Context,
	entry EntryModel,
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	if !typeutils.IsKnown(entry.Values) {
		diags.AddError("Invalid Configuration", "Attribute 'values' is required when type is 'match_any'")
		return result, diags
	}

	values := typeutils.ListTypeAs[string](ctx, entry.Values, path.Empty(), &diags)
	if diags.HasError() {
		return result, diags
	}

	if len(values) == 0 {
		diags.AddError("Invalid Configuration", "Attribute 'values' must contain at least one value when type is 'match_any'")
		return result, diags
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryMatchAny{
		Type:     EntryTypeMatchAny,
		Field:    field,
		Operator: operator,
		Value:    values,
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryMatchAny(apiEntry); err != nil {
		diags.AddError("Failed to create match_any entry", err.Error())
	}
	return result, diags
}

func convertListEntryToAPI(
	ctx context.Context,
	entry EntryModel,
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	if !typeutils.IsKnown(entry.List) {
		diags.AddError("Invalid Configuration", "Attribute 'list' is required when type is 'list'")
		return result, diags
	}

	var listModel EntryListModel
	diags.Append(entry.List.As(ctx, &listModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return result, diags
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryList{
		Type:     EntryTypeList,
		Field:    field,
		Operator: operator,
	}
	apiEntry.List.Id = listModel.ID.ValueString()
	apiEntry.List.Type = kbapi.SecurityExceptionsAPIListType(listModel.Type.ValueString())
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryList(apiEntry); err != nil {
		diags.AddError("Failed to create list entry", err.Error())
	}
	return result, diags
}

func convertExistsEntryToAPI(
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryExists{
		Type:     EntryTypeExists,
		Field:    field,
		Operator: operator,
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryExists(apiEntry); err != nil {
		diags.AddError("Failed to create exists entry", err.Error())
	}
	return result, diags
}

func convertWildcardEntryToAPI(
	entry EntryModel,
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	if !typeutils.IsKnown(entry.Value) || entry.Value.ValueString() == "" {
		diags.AddError("Invalid Configuration", "Attribute 'value' is required when type is 'wildcard'")
		return result, diags
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryMatchWildcard{
		Type:     "wildcard",
		Field:    field,
		Operator: operator,
		Value:    entry.Value.ValueString(),
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryMatchWildcard(apiEntry); err != nil {
		diags.AddError("Failed to create wildcard entry", err.Error())
	}
	return result, diags
}

func convertNestedEntryArrayToAPI(ctx context.Context, entry EntryModel, field kbapi.SecurityExceptionsAPINonEmptyString) (kbapi.SecurityExceptionsAPIExceptionListItemEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntry

	if !typeutils.IsKnown(entry.Entries) {
		diags.AddError("Invalid Configuration", "Attribute 'entries' is required when type is 'nested'")
		return result, diags
	}

	nestedEntries := typeutils.ListTypeAs[NestedEntryModel](ctx, entry.Entries, path.Empty(), &diags)
	if diags.HasError() {
		return result, diags
	}

	if len(nestedEntries) == 0 {
		diags.AddError("Invalid Configuration", "Attribute 'entries' must contain at least one entry when type is 'nested'")
		return result, diags
	}

	apiNestedEntries := make([]kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem, 0, len(nestedEntries))
	for _, ne := range nestedEntries {
		nestedAPIEntry, d := ConvertNestedEntryToAPI(ctx, ne)
		diags.Append(d...)
		if d.HasError() {
			continue
		}
		apiNestedEntries = append(apiNestedEntries, nestedAPIEntry)
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryNested{
		Type:    "nested",
		Field:   field,
		Entries: apiNestedEntries,
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryNested(apiEntry); err != nil {
		diags.AddError("Failed to create nested entry", err.Error())
	}
	return result, diags
}

// ConvertNestedEntryToAPI converts a nested entry model to an API nested entry.
func ConvertNestedEntryToAPI(ctx context.Context, entry NestedEntryModel) (kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem

	entryType := entry.Type.ValueString()
	operator := kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator(entry.Operator.ValueString())
	field := entry.Field.ValueString()

	switch entryType {
	case EntryTypeMatch:
		return convertNestedMatchEntryToAPI(entry, field, operator)
	case EntryTypeMatchAny:
		return convertNestedMatchAnyEntryToAPI(ctx, entry, field, operator)
	case EntryTypeExists:
		return convertNestedExistsEntryToAPI(field, operator)
	default:
		diags.AddError("Invalid nested entry type", fmt.Sprintf("Unknown nested entry type: %s. Only 'match', 'match_any', and 'exists' are allowed.", entryType))
		return result, diags
	}
}

func convertNestedMatchEntryToAPI(
	entry NestedEntryModel,
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem

	if !typeutils.IsKnown(entry.Value) || entry.Value.ValueString() == "" {
		diags.AddError("Invalid Configuration", "Attribute 'value' is required for nested entry when type is 'match'")
		return result, diags
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryMatch{
		Type:     EntryTypeMatch,
		Field:    field,
		Operator: operator,
		Value:    entry.Value.ValueString(),
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryMatch(apiEntry); err != nil {
		diags.AddError("Failed to create nested match entry", err.Error())
	}
	return result, diags
}

func convertNestedMatchAnyEntryToAPI(
	ctx context.Context,
	entry NestedEntryModel,
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem

	if !typeutils.IsKnown(entry.Values) {
		diags.AddError("Invalid Configuration", "Attribute 'values' is required for nested entry when type is 'match_any'")
		return result, diags
	}

	values := typeutils.ListTypeAs[string](ctx, entry.Values, path.Empty(), &diags)
	if diags.HasError() {
		return result, diags
	}

	if len(values) == 0 {
		diags.AddError("Invalid Configuration", "Attribute 'values' must contain at least one value for nested entry when type is 'match_any'")
		return result, diags
	}

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryMatchAny{
		Type:     EntryTypeMatchAny,
		Field:    field,
		Operator: operator,
		Value:    values,
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryMatchAny(apiEntry); err != nil {
		diags.AddError("Failed to create nested match_any entry", err.Error())
	}
	return result, diags
}

func convertNestedExistsEntryToAPI(
	field kbapi.SecurityExceptionsAPINonEmptyString,
	operator kbapi.SecurityExceptionsAPIExceptionListItemEntryOperator,
) (kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result kbapi.SecurityExceptionsAPIExceptionListItemEntryNestedEntryItem

	apiEntry := kbapi.SecurityExceptionsAPIExceptionListItemEntryExists{
		Type:     "exists",
		Field:    field,
		Operator: operator,
	}
	if err := result.FromSecurityExceptionsAPIExceptionListItemEntryExists(apiEntry); err != nil {
		diags.AddError("Failed to create nested exists entry", err.Error())
	}
	return result, diags
}

// ConvertEntriesFromAPI converts an API entry array to a Terraform entries list.
func ConvertEntriesFromAPI(ctx context.Context, apiEntries kbapi.SecurityExceptionsAPIExceptionListItemEntryArray) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(apiEntries) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: GetEntryAttrTypes()}), diags
	}

	entries := make([]EntryModel, 0, len(apiEntries))
	for _, apiEntry := range apiEntries {
		entry, d := ConvertEntryFromAPI(ctx, apiEntry)
		diags.Append(d...)
		if d.HasError() {
			continue
		}
		entries = append(entries, entry)
	}

	list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: GetEntryAttrTypes()}, entries)
	diags.Append(d...)
	return list, diags
}

// ConvertEntryFromAPI converts a single API entry to a Terraform entry model.
func ConvertEntryFromAPI(ctx context.Context, apiEntry kbapi.SecurityExceptionsAPIExceptionListItemEntry) (EntryModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var entry EntryModel

	entryBytes, err := apiEntry.MarshalJSON()
	if err != nil {
		diags.AddError("Failed to marshal entry", err.Error())
		return entry, diags
	}

	var entryMap map[string]any
	if err := json.Unmarshal(entryBytes, &entryMap); err != nil {
		diags.AddError("Failed to unmarshal entry", err.Error())
		return entry, diags
	}

	entryType, ok := entryMap["type"].(string)
	if !ok {
		diags.AddError("Invalid entry", "Entry is missing 'type' field")
		return entry, diags
	}

	entry.Type = types.StringValue(entryType)
	if field, ok := entryMap["field"].(string); ok {
		entry.Field = types.StringValue(field)
	}
	if operator, ok := entryMap["operator"].(string); ok {
		entry.Operator = types.StringValue(operator)
	}

	switch entryType {
	case EntryTypeMatch, EntryTypeWildcard:
		convertMatchOrWildcardEntryFromAPI(entryMap, &entry)
	case EntryTypeMatchAny:
		d := convertMatchAnyEntryFromAPI(ctx, entryMap, &entry)
		diags.Append(d...)
	case EntryTypeList:
		d := convertListEntryFromAPI(ctx, entryMap, &entry)
		diags.Append(d...)
	case EntryTypeExists:
		convertExistsEntryFromAPI(&entry)
	case EntryTypeNested:
		d := convertNestedEntryFromAPI(ctx, entryMap, &entry)
		diags.Append(d...)
	}

	return entry, diags
}

func convertMatchOrWildcardEntryFromAPI(entryMap map[string]any, entry *EntryModel) {
	if value, ok := entryMap["value"].(string); ok {
		entry.Value = types.StringValue(value)
	} else {
		entry.Value = types.StringNull()
	}
	entry.Values = types.ListNull(types.StringType)
	entry.List = types.ObjectNull(GetListAttrTypes())
	entry.Entries = types.ListNull(types.ObjectType{AttrTypes: GetNestedEntryAttrTypes()})
}

func convertMatchAnyEntryFromAPI(ctx context.Context, entryMap map[string]any, entry *EntryModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if values, ok := entryMap["value"].([]any); ok {
		strValues := make([]string, 0, len(values))
		for _, v := range values {
			if str, ok := v.(string); ok {
				strValues = append(strValues, str)
			}
		}
		list, d := types.ListValueFrom(ctx, types.StringType, strValues)
		diags.Append(d...)
		entry.Values = list
	} else {
		entry.Values = types.ListNull(types.StringType)
	}
	entry.Value = types.StringNull()
	entry.List = types.ObjectNull(GetListAttrTypes())
	entry.Entries = types.ListNull(types.ObjectType{AttrTypes: GetNestedEntryAttrTypes()})
	return diags
}

func convertListEntryFromAPI(ctx context.Context, entryMap map[string]any, entry *EntryModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if listData, ok := entryMap["list"].(map[string]any); ok {
		listModel := EntryListModel{
			ID:   types.StringValue(listData["id"].(string)),
			Type: types.StringValue(listData["type"].(string)),
		}
		obj, d := types.ObjectValueFrom(ctx, GetListAttrTypes(), listModel)
		diags.Append(d...)
		entry.List = obj
	} else {
		entry.List = types.ObjectNull(GetListAttrTypes())
	}
	entry.Value = types.StringNull()
	entry.Values = types.ListNull(types.StringType)
	entry.Entries = types.ListNull(types.ObjectType{AttrTypes: GetNestedEntryAttrTypes()})
	return diags
}

func convertExistsEntryFromAPI(entry *EntryModel) {
	entry.Value = types.StringNull()
	entry.Values = types.ListNull(types.StringType)
	entry.List = types.ObjectNull(GetListAttrTypes())
	entry.Entries = types.ListNull(types.ObjectType{AttrTypes: GetNestedEntryAttrTypes()})
}

func convertNestedEntryFromAPI(ctx context.Context, entryMap map[string]any, entry *EntryModel) diag.Diagnostics {
	var diags diag.Diagnostics

	entry.Operator = types.StringNull()
	if entriesData, ok := entryMap["entries"].([]any); ok {
		nestedEntries := make([]NestedEntryModel, 0, len(entriesData))
		for _, neData := range entriesData {
			if neMap, ok := neData.(map[string]any); ok {
				ne, d := convertNestedEntryFromMap(ctx, neMap)
				diags.Append(d...)
				if !d.HasError() {
					nestedEntries = append(nestedEntries, ne)
				}
			}
		}
		list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: GetNestedEntryAttrTypes()}, nestedEntries)
		diags.Append(d...)
		entry.Entries = list
	} else {
		entry.Entries = types.ListNull(types.ObjectType{AttrTypes: GetNestedEntryAttrTypes()})
	}
	entry.Value = types.StringNull()
	entry.Values = types.ListNull(types.StringType)
	entry.List = types.ObjectNull(GetListAttrTypes())
	return diags
}

func convertNestedEntryFromMap(ctx context.Context, entryMap map[string]any) (NestedEntryModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var entry NestedEntryModel

	if entryType, ok := entryMap["type"].(string); ok {
		entry.Type = types.StringValue(entryType)
	}
	if field, ok := entryMap["field"].(string); ok {
		entry.Field = types.StringValue(field)
	}
	if operator, ok := entryMap["operator"].(string); ok {
		entry.Operator = types.StringValue(operator)
	}

	switch entry.Type.ValueString() {
	case EntryTypeMatch:
		if value, ok := entryMap["value"].(string); ok {
			entry.Value = types.StringValue(value)
		} else {
			entry.Value = types.StringNull()
		}
		entry.Values = types.ListNull(types.StringType)
	case EntryTypeMatchAny:
		if values, ok := entryMap["value"].([]any); ok {
			strValues := make([]string, 0, len(values))
			for _, v := range values {
				if str, ok := v.(string); ok {
					strValues = append(strValues, str)
				}
			}
			list, d := types.ListValueFrom(ctx, types.StringType, strValues)
			diags.Append(d...)
			entry.Values = list
		} else {
			entry.Values = types.ListNull(types.StringType)
		}
		entry.Value = types.StringNull()
	case EntryTypeExists:
		entry.Value = types.StringNull()
		entry.Values = types.ListNull(types.StringType)
	}

	return entry, diags
}
