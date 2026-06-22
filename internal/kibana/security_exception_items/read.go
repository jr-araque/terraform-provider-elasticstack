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
		nsType = "single"
	}

	apiItems, d := kibanaoapi.FindExceptionListItemsAllPages(ctx, oapiClient, spaceID, listID, nsType)
	diags.Append(d...)
	if diags.HasError() {
		return model, false, diags
	}

	// Build a prior-state map by item_id for order stabilisation and empty-set preservation
	priorByItemID := make(map[string]*BulkItemModel)
	if priorItems, d := planItems(ctx, model); d == nil {
		for i := range priorItems {
			if !priorItems[i].ItemID.IsNull() && priorItems[i].ItemID.ValueString() != "" {
				id := priorItems[i].ItemID.ValueString()
				priorByItemID[id] = &priorItems[i]
			}
		}
	}

	// If list doesn't exist (no items and prior state had items), treat as deleted
	if len(apiItems) == 0 {
		if _, d := planItems(ctx, model); d == nil {
			if priorLen, _ := planItems(ctx, model); len(priorLen) > 0 {
				return model, false, diags
			}
		}
	}

	// Convert API items to model, preserving prior-state order
	converted := make([]BulkItemModel, 0, len(apiItems))
	apiByItemID := make(map[string]int, len(apiItems))
	for i, apiItem := range apiItems {
		apiByItemID[apiItem.ItemId] = i
	}

	// First: items that were in prior state (preserve order)
	for _, prior := range getPriorOrder(ctx, model) {
		idx, found := apiByItemID[prior.ItemID.ValueString()]
		if !found {
			continue
		}
		item, d := itemFromAPI(ctx, apiItems[idx], &prior)
		diags.Append(d...)
		converted = append(converted, item)
		delete(apiByItemID, apiItems[idx].ItemId)
	}

	// Then: new items not in prior state
	for itemID, idx := range apiByItemID {
		_ = itemID
		item, d := itemFromAPI(ctx, apiItems[idx], nil)
		diags.Append(d...)
		converted = append(converted, item)
	}

	d = setItems(ctx, &model, converted)
	diags.Append(d...)
	model.ID = types.StringValue(buildCompositeID(spaceID, listID))

	return model, true, diags
}

// getPriorOrder returns the prior state's items in their stored order.
func getPriorOrder(ctx context.Context, model ExceptionItemsModel) []BulkItemModel {
	items, _ := planItems(ctx, model)
	return items
}
