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

	// Transport (HTTP) errors are fatal — Kibana is unreachable or misconfigured
	// and continuing would fire further calls against a broken endpoint. Per-item
	// errors inside a successful HTTP response are accumulated so the caller sees
	// all failures in one plan output.
	//
	// The envelope always runs read-after-write after this returns, so the final
	// persisted state reflects actual live Kibana state regardless of which items
	// succeeded or failed.

	// Bulk create new items (chunked).
	for start := 0; start < len(toCreate); start += bulkChunkSize {
		chunk := toCreate[start:min(start+bulkChunkSize, len(toCreate))]

		reqItems := make([]kibanaoapi.ExceptionItemBulkCreateItemRequest, 0, len(chunk))
		for _, item := range chunk {
			r, d := itemToBulkCreateRequest(ctx, item)
			diags.Append(d...)
			if diags.HasError() {
				return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
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
			// Transport error — abort remaining phases.
			return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
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

	// Bulk update changed items (chunked).
	for start := 0; start < len(toUpdate); start += bulkChunkSize {
		chunk := toUpdate[start:min(start+bulkChunkSize, len(toUpdate))]

		reqItems := make([]kibanaoapi.ExceptionItemBulkUpdateItemRequest, 0, len(chunk))
		for _, item := range chunk {
			r, d := itemToBulkUpdateRequest(ctx, item)
			diags.Append(d...)
			if diags.HasError() {
				return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
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
			return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
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

	// Bulk delete removed items (chunked).
	for start := 0; start < len(toDeleteIDs); start += bulkChunkSize {
		chunk := toDeleteIDs[start:min(start+bulkChunkSize, len(toDeleteIDs))]

		_, d := kibanaoapi.BulkDeleteExceptionListItems(ctx, oapiClient, req.SpaceID, kibanaoapi.ExceptionItemBulkDeleteRequest{
			IDs:           chunk,
			NamespaceType: nsType,
		})
		diags.Append(d...)
		if diags.HasError() {
			return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
		}
	}

	// Return the plan model without populating Items. The envelope performs a
	// read-after-write that fetches actual live state from Kibana, so Items
	// does not need to be set here. Returning m (rather than a zero model)
	// ensures GetResourceID / GetSpaceID resolve correctly for the read call.
	return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
}
