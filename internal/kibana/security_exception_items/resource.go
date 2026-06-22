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
	"strings"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/entitycore"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = newExceptionItemsResource()
	_ resource.ResourceWithConfigure   = newExceptionItemsResource()
	_ resource.ResourceWithImportState = newExceptionItemsResource()
)

// ExceptionItemsResource is the bulk exception list items resource.
type ExceptionItemsResource struct {
	*entitycore.KibanaResource[ExceptionItemsModel]
}

func newExceptionItemsResource() *ExceptionItemsResource {
	return &ExceptionItemsResource{
		KibanaResource: entitycore.NewKibanaResource[ExceptionItemsModel](
			entitycore.ComponentKibana,
			"security_exception_items",
			entitycore.KibanaResourceOptions[ExceptionItemsModel]{
				Schema: getSchema,
				Read:   readExceptionItems,
				Delete: deleteExceptionItems,
				Create: createExceptionItems,
				Update: updateExceptionItems,
			},
		),
	}
}

// NewResource is a helper function to simplify the provider implementation.
func NewResource() resource.Resource {
	return newExceptionItemsResource()
}

// ImportState supports two import ID formats:
//
//   - "<spaceID>/<listID>" — namespace_type defaults to "single"
//   - "<spaceID>/<listID>/agnostic" — for space-agnostic lists
//
// The third segment, if present, is written to the namespace_type attribute
// so that Read uses the correct namespace when fetching items.
func (r *ExceptionItemsResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	// Write the raw import ID to the "id" attribute — the envelope's Read
	// callback will split it into spaceID + listID via CompositeIDFromStr.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)

	// If a third segment is present, extract namespace_type from it.
	// Format: "<spaceID>/<listID>/<namespaceType>"
	compID, diags := clients.CompositeIDFromStr(request.ID)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// CompositeIDFromStr only parses two segments. Check for a third manually.
	parts := strings.SplitN(request.ID, "/", 3)
	if len(parts) == 3 {
		nsType := parts[2]
		response.Diagnostics.Append(
			response.State.SetAttribute(ctx, path.Root("namespace_type"), types.StringValue(nsType))...,
		)
	}

	_ = compID // used implicitly via ImportStatePassthroughID above
}
