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
	"fmt"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibanaoapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/entitycore"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func updateExceptionItems(
	ctx context.Context,
	client *clients.KibanaScopedClient,
	req entitycore.KibanaWriteRequest[ExceptionItemsModel],
) (entitycore.KibanaWriteResult[ExceptionItemsModel], diag.Diagnostics) {
	m := req.Plan
	var diags diag.Diagnostics

	oapiClient := client.GetKibanaOapiClient()

	planItemsList, d := planItems(ctx, m)
	diags.Append(d...)
	if diags.HasError() {
		return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
	}

	var stateItemsList []BulkItemModel
	if req.Prior != nil {
		stateItemsList, d = planItems(ctx, *req.Prior)
		diags.Append(d...)
		if diags.HasError() {
			return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
		}
	}

	toCreate, toUpdate, toDeleteIDs := diffItems(planItemsList, stateItemsList)

	listID := m.ListID.ValueString()
	nsType := m.NamespaceType.ValueString()

	var resultItems []BulkItemModel

	// Bulk create new items (chunked)
	if len(toCreate) > 0 {
		for start := 0; start < len(toCreate); start += bulkChunkSize {
			end := min(start+bulkChunkSize, len(toCreate))
			chunk := toCreate[start:end]

			reqItems := make([]kibanaoapi.ExceptionItemBulkCreateItemRequest, 0, len(chunk))
			for _, item := range chunk {
				r, d := itemToBulkCreateRequest(ctx, item)
				diags.Append(d...)
				if diags.HasError() {
					return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
				}
				reqItems = append(reqItems, r)
			}

			bulkResp, d := kibanaoapi.BulkCreateExceptionListItems(ctx, oapiClient, req.SpaceID, kibanaoapi.ExceptionItemBulkCreateRequest{
				ListID:        listID,
				NamespaceType: nsType,
				Items:         reqItems,
			})
			diags.Append(d...)
			if diags.HasError() {
				return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
			}

			for _, apiItem := range bulkResp.Items {
				item, d := itemFromAPI(ctx, apiItem, nil)
				diags.Append(d...)
				resultItems = append(resultItems, item)
			}
			for _, e := range bulkResp.Errors {
				itemID := "(unknown)"
				if e.ItemID != nil {
					itemID = *e.ItemID
				}
				diags.AddError(
					fmt.Sprintf("Failed to create exception item %q during update", itemID),
					fmt.Sprintf("status %d: %s", e.Error.StatusCode, e.Error.Message),
				)
			}
		}
	}

	// Bulk update changed items (chunked)
	if len(toUpdate) > 0 {
		for start := 0; start < len(toUpdate); start += bulkChunkSize {
			end := min(start+bulkChunkSize, len(toUpdate))
			chunk := toUpdate[start:end]

			reqItems := make([]kibanaoapi.ExceptionItemBulkUpdateItemRequest, 0, len(chunk))
			for _, item := range chunk {
				r, d := itemToBulkUpdateRequest(ctx, item)
				diags.Append(d...)
				if diags.HasError() {
					return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
				}
				reqItems = append(reqItems, r)
			}

			bulkResp, d := kibanaoapi.BulkUpdateExceptionListItems(ctx, oapiClient, req.SpaceID, kibanaoapi.ExceptionItemBulkUpdateRequest{
				ListID:        listID,
				NamespaceType: nsType,
				Items:         reqItems,
			})
			diags.Append(d...)
			if diags.HasError() {
				return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
			}

			for _, apiItem := range bulkResp.Items {
				item, d := itemFromAPI(ctx, apiItem, nil)
				diags.Append(d...)
				resultItems = append(resultItems, item)
			}
			for _, e := range bulkResp.Errors {
				itemID := "(unknown)"
				if e.ItemID != nil {
					itemID = *e.ItemID
				}
				diags.AddError(
					fmt.Sprintf("Failed to update exception item %q", itemID),
					fmt.Sprintf("status %d: %s", e.Error.StatusCode, e.Error.Message),
				)
			}
		}
	}

	// Bulk delete removed items (chunked)
	if len(toDeleteIDs) > 0 {
		for start := 0; start < len(toDeleteIDs); start += bulkChunkSize {
			end := min(start+bulkChunkSize, len(toDeleteIDs))
			chunk := toDeleteIDs[start:end]

			_, d := kibanaoapi.BulkDeleteExceptionListItems(ctx, oapiClient, req.SpaceID, kibanaoapi.ExceptionItemBulkDeleteRequest{
				IDs:           chunk,
				NamespaceType: nsType,
			})
			diags.Append(d...)
			if diags.HasError() {
				return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
			}
		}
	}

	if diags.HasError() {
		return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
	}

	d = setItems(ctx, &m, resultItems)
	diags.Append(d...)

	return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
}
