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

// parseImportID splits a raw import ID into the canonical 2-segment composite ID
// and an optional namespace_type suffix. Detection uses HasSuffix rather than a
// positional split because list_id may itself contain forward slashes, and
// "agnostic" / "single" cannot appear as valid list_id segments.
func parseImportID(raw string) (id, nsType string) {
	switch {
	case strings.HasSuffix(raw, "/agnostic"):
		return strings.TrimSuffix(raw, "/agnostic"), "agnostic"
	case strings.HasSuffix(raw, "/single"):
		return strings.TrimSuffix(raw, "/single"), "single"
	default:
		return raw, ""
	}
}

// ImportState supports two import ID formats:
//
//   - "<spaceID>/<listID>"            — namespace_type defaults to "single"
//   - "<spaceID>/<listID>/agnostic"   — for space-agnostic lists
//   - "<spaceID>/<listID>/single"     — explicit single (same as default)
//
// namespace_type is detected as a suffix rather than a positional split because
// list_id may itself contain forward slashes. The suffix approach is unambiguous
// since the only valid values ("single", "agnostic") cannot collide with a list_id.
func (r *ExceptionItemsResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	id, nsType := parseImportID(request.ID)

	if id == "" {
		response.Diagnostics.AddError(
			"Invalid import ID",
			"Expected format: <space_id>/<list_id> or <space_id>/<list_id>/agnostic",
		)
		return
	}

	response.Diagnostics.Append(
		response.State.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...,
	)

	if nsType != "" {
		response.Diagnostics.Append(
			response.State.SetAttribute(ctx, path.Root("namespace_type"), types.StringValue(nsType))...,
		)
	}
}
