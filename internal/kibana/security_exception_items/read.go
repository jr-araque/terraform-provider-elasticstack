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
	"context"
	"sort"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibanaoapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func readExceptionItems(
	ctx context.Context,
	client *clients.KibanaScopedClient,
	resourceID, spaceID string,
	model ExceptionItemsModel,
) (ExceptionItemsModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	oapiClient := client.GetKibanaOapiClient()
	model.SpaceID = types.StringValue(spaceID)

	listID := resourceID
	nsType := model.NamespaceType.ValueString()
	if nsType == "" {
		nsType = namespaceTypeSingle
	}
	nsTypeVal := kbapi.SecurityExceptionsAPIExceptionNamespaceType(nsType)

	// Check that the parent exception list still exists. An empty items result
	// from _find is ambiguous: it could mean the list exists with 0 items, or
	// the list was deleted out-of-band. We disambiguate with an explicit GET.
	list, d := kibanaoapi.GetExceptionList(ctx, oapiClient, spaceID, &kbapi.ReadExceptionListParams{
		ListId:        &listID,
		NamespaceType: &nsTypeVal,
	})
	diags.Append(d...)
	if diags.HasError() {
		return model, false, diags
	}
	if list == nil {
		// 404 — the list itself is gone; remove the resource from state.
		return model, false, diags
	}

	apiItems, d := kibanaoapi.FindExceptionListItemsAllPages(ctx, oapiClient, spaceID, listID, nsType)
	diags.Append(d...)
	if diags.HasError() {
		return model, false, diags
	}

	// Build a lookup of API items by item_id.
	apiByItemID := make(map[string]int, len(apiItems))
	for i, apiItem := range apiItems {
		apiByItemID[apiItem.ItemId] = i
	}

	converted := make([]BulkItemModel, 0, len(apiItems))

	// First: items that were in prior state, in their stored order.
	// This preserves the user's declared order across refreshes.
	priorItems, pd := planItems(ctx, model)
	diags.Append(pd...)
	for _, prior := range priorItems {
		idx, found := apiByItemID[prior.ItemID.ValueString()]
		if !found {
			continue
		}
		item, d := itemFromAPI(ctx, apiItems[idx], &prior)
		diags.Append(d...)
		converted = append(converted, item)
		delete(apiByItemID, apiItems[idx].ItemId)
	}

	// Then: items not in prior state (e.g. imported resource, out-of-band additions).
	// Sort by item_id for deterministic ordering so repeated Reads produce identical
	// state and do not trigger spurious plan diffs.
	remaining := make([]string, 0, len(apiByItemID))
	for itemID := range apiByItemID {
		remaining = append(remaining, itemID)
	}
	sort.Strings(remaining)
	for _, itemID := range remaining {
		idx := apiByItemID[itemID]
		item, d := itemFromAPI(ctx, apiItems[idx], nil)
		diags.Append(d...)
		converted = append(converted, item)
	}

	model.NamespaceType = types.StringValue(nsType)
	d = setItems(ctx, &model, converted)
	diags.Append(d...)
	model.ID = types.StringValue(buildCompositeID(spaceID, listID))

	return model, true, diags
}
