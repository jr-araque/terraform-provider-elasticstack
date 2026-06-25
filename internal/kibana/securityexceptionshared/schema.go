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

package securityexceptionshared

import (
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/validators"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EntriesSchema returns the schema for the entries list attribute, shared
// between the single-item and bulk exception item resources.
func EntriesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "The exception item entries. This defines the conditions under which the exception applies.",
		Required:            true,
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				AttrType: schema.StringAttribute{
					MarkdownDescription: "The type of entry. Valid values: `match`, `match_any`, `list`, `exists`, `nested`, `wildcard`.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(EntryTypeMatch, EntryTypeMatchAny, EntryTypeList, EntryTypeExists, EntryTypeNested, EntryTypeWildcard),
					},
				},
				AttrField: schema.StringAttribute{
					MarkdownDescription: "The field name. Required for all entry types.",
					Required:            true,
				},
				AttrOperator: schema.StringAttribute{
					MarkdownDescription: "The operator to use. Valid values: `included`, `excluded`. Note: The operator field is not supported for nested entry types and will be ignored if specified.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("included", "excluded"),
					},
				},
				AttrValue: schema.StringAttribute{
					MarkdownDescription: "The value to match (for `match` and `wildcard` types).",
					Optional:            true,
					Validators: []validator.String{
						validators.RequiredIfDependentPathExpressionOneOf(
							path.MatchRelative().AtParent().AtName(AttrType),
							[]string{EntryTypeMatch, EntryTypeWildcard},
						),
					},
				},
				AttrValues: schema.ListAttribute{
					ElementType:         types.StringType,
					MarkdownDescription: "Array of values to match (for `match_any` type).",
					Optional:            true,
				},
				EntryTypeList: schema.SingleNestedAttribute{
					MarkdownDescription: "Value list reference (for `list` type).",
					Optional:            true,
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The value list ID.",
							Required:            true,
						},
						AttrType: schema.StringAttribute{
							MarkdownDescription: "The value list type (e.g., `keyword`, `ip`, `ip_range`).",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("keyword", "ip", "ip_range"),
							},
						},
					},
				},
				AttrEntries: schema.ListNestedAttribute{
					MarkdownDescription: "Nested entries (for `nested` type). Only `match`, `match_any`, and `exists` entry types are allowed as nested entries.",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							AttrType: schema.StringAttribute{
								MarkdownDescription: "The type of nested entry. Valid values: `match`, `match_any`, `exists`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf(EntryTypeMatch, EntryTypeMatchAny, EntryTypeExists),
								},
							},
							AttrField: schema.StringAttribute{
								MarkdownDescription: "The field name.",
								Required:            true,
							},
							AttrOperator: schema.StringAttribute{
								MarkdownDescription: "The operator to use. Valid values: `included`, `excluded`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("included", "excluded"),
								},
							},
							AttrValue: schema.StringAttribute{
								MarkdownDescription: "The value to match (for `match` type).",
								Optional:            true,
								Validators: []validator.String{
									validators.RequiredIfDependentPathExpressionOneOf(
										path.MatchRelative().AtParent().AtName(AttrType),
										[]string{EntryTypeMatch},
									),
								},
							},
							AttrValues: schema.ListAttribute{
								ElementType:         types.StringType,
								MarkdownDescription: "Array of values to match (for `match_any` type).",
								Optional:            true,
								Validators: []validator.List{
									validators.RequiredIfDependentPathExpressionOneOf(
										path.MatchRelative().AtParent().AtName(AttrType),
										[]string{EntryTypeMatchAny},
									),
								},
							},
						},
					},
				},
			},
		},
	}
}

// CommentsSchema returns the schema for the comments list attribute.
// Computed is true for the comment ID (auto-assigned by Kibana).
func CommentsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Array of comments about the exception item.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "The unique identifier of the comment (auto-generated by Kibana).",
					Computed:            true,
				},
				"comment": schema.StringAttribute{
					MarkdownDescription: "The comment text.",
					Required:            true,
				},
			},
		},
	}
}
