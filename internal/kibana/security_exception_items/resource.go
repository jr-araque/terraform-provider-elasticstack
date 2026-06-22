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
//   - "<spaceID>/<listID>"           — namespace_type defaults to "single"
//   - "<spaceID>/<listID>/agnostic"  — for space-agnostic lists
//
// The canonical composite ID stored in state is always the 2-segment form
// "<spaceID>/<listID>". The third segment, when present, is written to
// namespace_type and stripped from the id attribute so that the envelope's
// CompositeIDFromStr parses a clean spaceID + listID pair.
func (r *ExceptionItemsResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	parts := strings.SplitN(request.ID, "/", 3)
	if len(parts) < 2 {
		response.Diagnostics.AddError(
			"Invalid import ID",
			"Expected format: <space_id>/<list_id> or <space_id>/<list_id>/agnostic",
		)
		return
	}

	// Store only the 2-segment composite so CompositeIDFromStr in Read
	// resolves spaceID=parts[0] and listID=parts[1] without the third segment
	// being folded into the listID.
	canonicalID := parts[0] + "/" + parts[1]
	response.Diagnostics.Append(
		response.State.SetAttribute(ctx, path.Root("id"), types.StringValue(canonicalID))...,
	)

	if len(parts) == 3 {
		response.Diagnostics.Append(
			response.State.SetAttribute(ctx, path.Root("namespace_type"), types.StringValue(parts[2]))...,
		)
	}
}
