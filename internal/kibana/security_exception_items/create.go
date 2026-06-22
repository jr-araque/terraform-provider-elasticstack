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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// bulkChunkSize is the maximum number of items per bulk API request.
const bulkChunkSize = 1000

func createExceptionItems(
	ctx context.Context,
	client *clients.KibanaScopedClient,
	req entitycore.KibanaWriteRequest[ExceptionItemsModel],
) (entitycore.KibanaWriteResult[ExceptionItemsModel], diag.Diagnostics) {
	m := req.Plan
	var diags diag.Diagnostics

	oapiClient := client.GetKibanaOapiClient()

	items, d := planItems(ctx, m)
	diags.Append(d...)
	if diags.HasError() {
		return entitycore.KibanaWriteResult[ExceptionItemsModel]{}, diags
	}

	listID := m.ListID.ValueString()
	nsType := m.NamespaceType.ValueString()

	for start := 0; start < len(items); start += bulkChunkSize {
		chunk := items[start:min(start+bulkChunkSize, len(items))]

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
			// Transport error — set the composite ID so the envelope's
			// read-after-write can still locate the partially-created resource.
			m.ID = types.StringValue(buildCompositeID(req.SpaceID, listID))
			return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
		}

		for _, e := range bulkResp.Errors {
			itemID := "(unknown)"
			if e.ItemID != nil {
				itemID = *e.ItemID
			}
			diags.AddError(
				fmt.Sprintf("Failed to create exception item %q", itemID),
				fmt.Sprintf("status %d: %s", e.Error.StatusCode, e.Error.Message),
			)
		}
	}

	// Set the composite ID so resolveKibanaResourceIdentity can derive spaceID
	// and listID for the mandatory read-after-write. Items are not set here;
	// the envelope's Read call provides the authoritative state.
	m.ID = types.StringValue(buildCompositeID(req.SpaceID, listID))
	return entitycore.KibanaWriteResult[ExceptionItemsModel]{Model: m}, diags
}
