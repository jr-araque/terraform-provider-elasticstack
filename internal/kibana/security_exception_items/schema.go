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

	shared "github.com/elastic/terraform-provider-elasticstack/internal/kibana/securityexceptionshared"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// maxBulkItems is the maximum number of items managed by this resource,
// matching the Kibana exception list capacity.
const maxBulkItems = 10000

func getSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages **all** exception list items for a given exception list as a single " +
			"aggregate resource. Supports up to 10,000 items; all bulk API calls are internally chunked " +
			"into 1,000-item batches.\n\n" +
			"~> **Ownership conflict**: Do not mix this resource with " +
			"`elasticstack_kibana_security_exception_item` (singular) for the same `list_id`. " +
			"This resource's Read fetches all items from the list — on the next apply it will " +
			"delete any items it does not recognise from its own state, including items managed " +
			"by the singular resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier of the resource (`<space_id>/<list_id>`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"space_id": schema.StringAttribute{
				MarkdownDescription: "An identifier for the Kibana space. Defaults to `default`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"list_id": schema.StringAttribute{
				MarkdownDescription: "The human-readable identifier of the exception list that contains these items.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace_type": schema.StringAttribute{
				MarkdownDescription: "Determines whether the exception list is available in all Kibana spaces or just the current space. Can be `single` (default) or `agnostic`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(namespaceTypeSingle),
				Validators: []validator.String{
					stringvalidator.OneOf(namespaceTypeSingle, namespaceTypeAgnostic),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"items": schema.ListNestedAttribute{
				MarkdownDescription: "The exception items to manage. Each item corresponds to one exception condition in the list.",
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, maxBulkItems),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: itemAttributes(),
				},
			},
		},
	}
}

func itemAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the exception item (Kibana document ID, auto-generated).",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"item_id": schema.StringAttribute{
			MarkdownDescription: "The human-readable identifier for this exception item. " +
				"Required — used as the stable diff key to match plan items to state items " +
				"across updates. Without a stable `item_id`, adding or removing other items " +
				"in the list causes incorrect update/delete assignments.",
			Required: true,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "The type of exception item. Must be `simple`.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("simple"),
			},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the exception item.",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "A description of the exception item.",
			Required:            true,
		},
		shared.AttrEntries: shared.EntriesSchema(),
		"os_types": schema.SetAttribute{
			MarkdownDescription: "OS types this exception applies to. Valid values: `linux`, `macos`, `windows`.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"tags": schema.SetAttribute{
			MarkdownDescription: "Tags for categorising this exception item.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"meta": schema.StringAttribute{
			MarkdownDescription: "Arbitrary metadata for this exception item as a JSON string.",
			Optional:            true,
			CustomType:          jsontypes.NormalizedType{},
		},
		"expire_time": schema.StringAttribute{
			MarkdownDescription: "The exception item's expiration date in RFC3339 format.",
			Optional:            true,
			Computed:            true,
			CustomType:          timetypes.RFC3339Type{},
		},
		"comments": shared.CommentsSchema(),
		"_version": schema.StringAttribute{
			MarkdownDescription: "The version token used for optimistic concurrency control during updates.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tie_breaker_id": schema.StringAttribute{
			MarkdownDescription: "Field used to ensure stable sort order for this exception item.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "Timestamp when the exception item was created.",
			Computed:            true,
		},
		"created_by": schema.StringAttribute{
			MarkdownDescription: "User who created the exception item.",
			Computed:            true,
		},
		"updated_at": schema.StringAttribute{
			MarkdownDescription: "Timestamp when the exception item was last updated.",
			Computed:            true,
		},
		"updated_by": schema.StringAttribute{
			MarkdownDescription: "User who last updated the exception item.",
			Computed:            true,
		},
	}
}
