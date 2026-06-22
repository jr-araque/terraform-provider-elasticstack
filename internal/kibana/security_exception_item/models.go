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

package securityexceptionitem

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/diagutil"
	"github.com/elastic/terraform-provider-elasticstack/internal/entitycore"
	shared "github.com/elastic/terraform-provider-elasticstack/internal/kibana/securityexceptionshared"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Re-export shared types so existing code in this package can use the short names.
type EntryModel = shared.EntryModel
type EntryListModel = shared.EntryListModel
type NestedEntryModel = shared.NestedEntryModel
type CommentModel = shared.CommentModel

// MinVersionExpireTime defines the minimum server version required for expire_time field
var MinVersionExpireTime = version.Must(version.NewVersion("8.7.2"))

type ExceptionItemModel struct {
	entitycore.ResourceTimeoutsField
	ID               types.String         `tfsdk:"id"`
	KibanaConnection types.List           `tfsdk:"kibana_connection"`
	SpaceID          types.String         `tfsdk:"space_id"`
	ItemID           types.String         `tfsdk:"item_id"`
	ListID           types.String         `tfsdk:"list_id"`
	Name             types.String         `tfsdk:"name"`
	Description      types.String         `tfsdk:"description"`
	Type             types.String         `tfsdk:"type"`
	NamespaceType    types.String         `tfsdk:"namespace_type"`
	OsTypes          types.Set            `tfsdk:"os_types"`
	Tags             types.Set            `tfsdk:"tags"`
	Meta             jsontypes.Normalized `tfsdk:"meta"`
	Entries          types.List           `tfsdk:"entries"`
	Comments         types.List           `tfsdk:"comments"`
	ExpireTime       timetypes.RFC3339    `tfsdk:"expire_time"`
	CreatedAt        types.String         `tfsdk:"created_at"`
	CreatedBy        types.String         `tfsdk:"created_by"`
	UpdatedAt        types.String         `tfsdk:"updated_at"`
	UpdatedBy        types.String         `tfsdk:"updated_by"`
	TieBreakerID     types.String         `tfsdk:"tie_breaker_id"`
}

func (m ExceptionItemModel) GetID() types.String { return m.ID }
func (m ExceptionItemModel) GetResourceID() types.String {
	if compID, _ := clients.CompositeIDFromStr(m.ID.ValueString()); compID != nil {
		return types.StringValue(compID.ResourceID)
	}
	return m.ItemID
}
func (m ExceptionItemModel) GetSpaceID() types.String        { return m.SpaceID }
func (m ExceptionItemModel) GetKibanaConnection() types.List { return m.KibanaConnection }

var _ entitycore.KibanaResourceModel = ExceptionItemModel{}

func (m ExceptionItemModel) GetVersionRequirements(_ context.Context) ([]entitycore.VersionRequirement, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !typeutils.IsKnown(m.ExpireTime) {
		return nil, diags
	}
	return []entitycore.VersionRequirement{{
		MinVersion:   *MinVersionExpireTime,
		ErrorMessage: fmt.Sprintf("expire_time requires server version %s or higher", MinVersionExpireTime.String()),
	}}, diags
}

var _ entitycore.WithVersionRequirements = ExceptionItemModel{}

// CommonExceptionItemProps holds pointers to common fields across create/update requests
type CommonExceptionItemProps struct {
	NamespaceType *kbapi.SecurityExceptionsAPIExceptionNamespaceType
	OsTypes       *[]kbapi.SecurityExceptionsAPIExceptionListOsType
	Tags          *kbapi.SecurityExceptionsAPIExceptionListItemTags
	Meta          *kbapi.SecurityExceptionsAPIExceptionListItemMeta
	ExpireTime    *kbapi.SecurityExceptionsAPIExceptionListItemExpireTime
}

// setCommonProps sets common optional fields on create and update requests.
func (m *ExceptionItemModel) setCommonProps(
	ctx context.Context,
	props *CommonExceptionItemProps,
	diags *diag.Diagnostics,
) {
	if typeutils.IsKnown(m.NamespaceType) {
		nsType := kbapi.SecurityExceptionsAPIExceptionNamespaceType(m.NamespaceType.ValueString())
		*props.NamespaceType = nsType
	}

	if typeutils.IsKnown(m.OsTypes) {
		osTypes := typeutils.SetTypeAs[kbapi.SecurityExceptionsAPIExceptionListOsType](ctx, m.OsTypes, path.Empty(), diags)
		if diags.HasError() {
			return
		}
		*props.OsTypes = osTypes
	}

	if typeutils.IsKnown(m.Tags) {
		tags := typeutils.SetTypeAs[string](ctx, m.Tags, path.Empty(), diags)
		if diags.HasError() {
			return
		}
		*props.Tags = tags
	}

	if typeutils.IsKnown(m.Meta) {
		var meta kbapi.SecurityExceptionsAPIExceptionListItemMeta
		unmarshalDiags := m.Meta.Unmarshal(&meta)
		diags.Append(unmarshalDiags...)
		if diags.HasError() {
			return
		}
		*props.Meta = meta
	}

	if typeutils.IsKnown(m.ExpireTime) {
		expireTime, d := m.ExpireTime.ValueRFC3339Time()
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		expireTimeAPI := expireTime.Format("2006-01-02T15:04:05.000Z")
		*props.ExpireTime = expireTimeAPI
	}
}

func (m *ExceptionItemModel) commentModels(ctx context.Context, diags *diag.Diagnostics) []CommentModel {
	if !typeutils.IsKnown(m.Comments) {
		return nil
	}
	comments := typeutils.ListTypeAs[CommentModel](ctx, m.Comments, path.Empty(), diags)
	if diags.HasError() || len(comments) == 0 {
		return nil
	}
	return comments
}

// toCreateRequest converts the Terraform model to an API create request.
func (m *ExceptionItemModel) toCreateRequest(ctx context.Context) (*kbapi.CreateExceptionListItemJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	entries, d := shared.ConvertEntriesToAPI(ctx, m.Entries)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	genericReq := kbapi.SecurityExceptionsAPICreateExceptionListItemGeneric{
		ListId:      m.ListID.ValueString(),
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		Type:        kbapi.SecurityExceptionsAPIExceptionListItemType(m.Type.ValueString()),
		Entries:     entries,
	}

	if typeutils.IsKnown(m.ItemID) {
		itemID := m.ItemID.ValueString()
		genericReq.ItemId = &itemID
	}

	var nsType kbapi.SecurityExceptionsAPIExceptionNamespaceType
	var osTypes []kbapi.SecurityExceptionsAPIExceptionListOsType
	var tags kbapi.SecurityExceptionsAPIExceptionListItemTags
	var meta kbapi.SecurityExceptionsAPIExceptionListItemMeta
	var expireTime kbapi.SecurityExceptionsAPIExceptionListItemExpireTime

	m.setCommonProps(ctx, &CommonExceptionItemProps{
		NamespaceType: &nsType,
		OsTypes:       &osTypes,
		Tags:          &tags,
		Meta:          &meta,
		ExpireTime:    &expireTime,
	}, &diags)
	if diags.HasError() {
		return nil, diags
	}

	if typeutils.IsKnown(m.NamespaceType) {
		genericReq.NamespaceType = &nsType
	}
	if typeutils.IsKnown(m.OsTypes) {
		genericReq.OsTypes = &osTypes
	}
	if typeutils.IsKnown(m.Tags) {
		genericReq.Tags = &tags
	}
	if typeutils.IsKnown(m.Meta) {
		genericReq.Meta = &meta
	}
	if typeutils.IsKnown(m.ExpireTime) {
		genericReq.ExpireTime = &expireTime
	}

	if comments := m.commentModels(ctx, &diags); comments != nil {
		commentsArray := make(kbapi.SecurityExceptionsAPICreateExceptionListItemCommentArray, len(comments))
		for i, comment := range comments {
			commentsArray[i] = kbapi.SecurityExceptionsAPICreateExceptionListItemComment{
				Comment: comment.Comment.ValueString(),
			}
		}
		genericReq.Comments = &commentsArray
	}
	if diags.HasError() {
		return nil, diags
	}

	req := &kbapi.CreateExceptionListItemJSONRequestBody{}
	err := req.FromSecurityExceptionsAPICreateExceptionListItemGeneric(genericReq)
	if err != nil {
		diags.Append(diagutil.FrameworkDiagFromError(err)...)
		return nil, diags
	}

	return req, diags
}

// toUpdateRequest converts the Terraform model to an API update request.
func (m *ExceptionItemModel) toUpdateRequest(ctx context.Context, resourceID string) (*kbapi.UpdateExceptionListItemJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	entries, d := shared.ConvertEntriesToAPI(ctx, m.Entries)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	id := resourceID
	genericReq := kbapi.SecurityExceptionsAPIUpdateExceptionListItemGeneric{
		Id:          &id,
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		Type:        kbapi.SecurityExceptionsAPIExceptionListItemType(m.Type.ValueString()),
		Entries:     entries,
	}

	var nsType kbapi.SecurityExceptionsAPIExceptionNamespaceType
	var osTypes []kbapi.SecurityExceptionsAPIExceptionListOsType
	var tags kbapi.SecurityExceptionsAPIExceptionListItemTags
	var meta kbapi.SecurityExceptionsAPIExceptionListItemMeta
	var expireTime kbapi.SecurityExceptionsAPIExceptionListItemExpireTime

	m.setCommonProps(ctx, &CommonExceptionItemProps{
		NamespaceType: &nsType,
		OsTypes:       &osTypes,
		Tags:          &tags,
		Meta:          &meta,
		ExpireTime:    &expireTime,
	}, &diags)
	if diags.HasError() {
		return nil, diags
	}

	if typeutils.IsKnown(m.NamespaceType) {
		genericReq.NamespaceType = &nsType
	}
	if typeutils.IsKnown(m.OsTypes) {
		genericReq.OsTypes = &osTypes
	}
	if typeutils.IsKnown(m.Tags) {
		genericReq.Tags = &tags
	}
	if typeutils.IsKnown(m.Meta) {
		genericReq.Meta = &meta
	}
	if typeutils.IsKnown(m.ExpireTime) {
		genericReq.ExpireTime = &expireTime
	}

	if comments := m.commentModels(ctx, &diags); comments != nil {
		commentsArray := make(kbapi.SecurityExceptionsAPIUpdateExceptionListItemCommentArray, len(comments))
		for i, comment := range comments {
			commentsArray[i] = kbapi.SecurityExceptionsAPIUpdateExceptionListItemComment{
				Comment: comment.Comment.ValueString(),
			}
		}
		genericReq.Comments = &commentsArray
	}
	if diags.HasError() {
		return nil, diags
	}

	req := &kbapi.UpdateExceptionListItemJSONRequestBody{}
	err := req.FromSecurityExceptionsAPIUpdateExceptionListItemGeneric(genericReq)
	if err != nil {
		diags.Append(diagutil.FrameworkDiagFromError(err)...)
		return nil, diags
	}

	return req, diags
}

// fromAPI converts the API response to Terraform model.
func (m *ExceptionItemModel) fromAPI(ctx context.Context, apiResp *kbapi.SecurityExceptionsAPIExceptionListItem) diag.Diagnostics {
	var diags diag.Diagnostics

	compID := clients.CompositeID{
		ClusterID:  m.SpaceID.ValueString(),
		ResourceID: typeutils.StringishValue(apiResp.Id).ValueString(),
	}
	m.ID = types.StringValue(compID.String())

	m.ItemID = typeutils.StringishValue(apiResp.ItemId)
	m.ListID = typeutils.StringishValue(apiResp.ListId)
	m.Name = typeutils.StringishValue(apiResp.Name)
	m.Description = typeutils.StringishValue(apiResp.Description)
	m.Type = typeutils.StringishValue(apiResp.Type)
	m.NamespaceType = typeutils.StringishValue(apiResp.NamespaceType)
	m.CreatedAt = types.StringValue(apiResp.CreatedAt.Format("2006-01-02T15:04:05.000Z"))
	m.CreatedBy = types.StringValue(apiResp.CreatedBy)
	m.UpdatedAt = types.StringValue(apiResp.UpdatedAt.Format("2006-01-02T15:04:05.000Z"))
	m.UpdatedBy = types.StringValue(apiResp.UpdatedBy)
	m.TieBreakerID = types.StringValue(apiResp.TieBreakerId)

	if apiResp.ExpireTime != nil {
		expireTime, err := time.Parse(time.RFC3339, *apiResp.ExpireTime)
		if err != nil {
			diags.AddError("Failed to parse expire_time from API response", err.Error())
			m.ExpireTime = timetypes.NewRFC3339Null()
		} else {
			m.ExpireTime = timetypes.NewRFC3339TimeValue(expireTime)
		}
	} else {
		m.ExpireTime = timetypes.NewRFC3339Null()
	}

	if apiResp.OsTypes != nil && len(*apiResp.OsTypes) > 0 {
		set, d := types.SetValueFrom(ctx, types.StringType, *apiResp.OsTypes)
		diags.Append(d...)
		m.OsTypes = set
	} else if m.OsTypes.IsUnknown() {
		m.OsTypes = types.SetNull(types.StringType)
	}

	if apiResp.Tags != nil && len(*apiResp.Tags) > 0 {
		set, d := types.SetValueFrom(ctx, types.StringType, *apiResp.Tags)
		diags.Append(d...)
		m.Tags = set
	} else if m.Tags.IsUnknown() {
		m.Tags = types.SetNull(types.StringType)
	}

	if apiResp.Meta != nil {
		metaBytes, err := json.Marshal(apiResp.Meta)
		if err != nil {
			diags.AddError("Failed to marshal meta field from API response to JSON", err.Error())
			return diags
		}
		m.Meta = jsontypes.NewNormalizedValue(string(metaBytes))
	} else {
		m.Meta = jsontypes.NewNormalizedNull()
	}

	entriesList, d := shared.ConvertEntriesFromAPI(ctx, apiResp.Entries)
	diags.Append(d...)
	m.Entries = entriesList

	if len(apiResp.Comments) > 0 {
		comments := make([]CommentModel, len(apiResp.Comments))
		for i, comment := range apiResp.Comments {
			comments[i] = CommentModel{
				ID:      typeutils.StringishValue(comment.Id),
				Comment: typeutils.StringishValue(comment.Comment),
			}
		}
		list, d := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: shared.GetCommentAttrTypes(),
		}, comments)
		diags.Append(d...)
		m.Comments = list
	} else {
		m.Comments = types.ListNull(types.ObjectType{
			AttrTypes: shared.GetCommentAttrTypes(),
		})
	}

	return diags
}
