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
)

func deleteExceptionItems(
	ctx context.Context,
	client *clients.KibanaScopedClient,
	resourceID, spaceID string,
	model ExceptionItemsModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	oapiClient := client.GetKibanaOapiClient()

	items, d := planItems(ctx, model)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	// Collect all item IDs from state
	var ids []string
	for _, item := range items {
		if !item.ID.IsNull() && item.ID.ValueString() != "" {
			ids = append(ids, item.ID.ValueString())
		}
	}

	if len(ids) == 0 {
		return diags
	}

	nsType := model.NamespaceType.ValueString()
	if nsType == "" {
		nsType = "single"
	}

	// Chunk deletes
	for start := 0; start < len(ids); start += bulkChunkSize {
		end := min(start+bulkChunkSize, len(ids))
		chunk := ids[start:end]

		_, d := kibanaoapi.BulkDeleteExceptionListItems(ctx, oapiClient, spaceID, kibanaoapi.ExceptionItemBulkDeleteRequest{
			IDs:           chunk,
			NamespaceType: nsType,
		})
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
	}

	return diags
}
