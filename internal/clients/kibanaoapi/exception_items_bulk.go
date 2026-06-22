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

package kibanaoapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibanautil"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ExceptionItemBulkCreateItemRequest is the per-item payload for a bulk create.
type ExceptionItemBulkCreateItemRequest struct {
	ItemID      *string                                                  `json:"item_id,omitempty"`
	Type        string                                                   `json:"type"`
	Name        string                                                   `json:"name"`
	Description string                                                   `json:"description"`
	Entries     kbapi.SecurityExceptionsAPIExceptionListItemEntryArray   `json:"entries"`
	OsTypes     []string                                                 `json:"os_types,omitempty"`
	Tags        []string                                                 `json:"tags,omitempty"`
	Meta        map[string]any                                           `json:"meta,omitempty"`
	ExpireTime  *string                                                  `json:"expire_time,omitempty"`
	Comments    []ExceptionItemBulkComment                               `json:"comments,omitempty"`
}

// ExceptionItemBulkComment is a comment payload for bulk create.
type ExceptionItemBulkComment struct {
	Comment string `json:"comment"`
}

// ExceptionItemBulkCreateRequest is the request body for bulk create.
type ExceptionItemBulkCreateRequest struct {
	ListID        string                                `json:"list_id"`
	NamespaceType string                                `json:"namespace_type,omitempty"`
	Items         []ExceptionItemBulkCreateItemRequest  `json:"items"`
}

// ExceptionItemBulkUpdateItemRequest is the per-item payload for a bulk update.
type ExceptionItemBulkUpdateItemRequest struct {
	ID          string                                                   `json:"id"`
	Version     *string                                                  `json:"_version,omitempty"`
	Type        string                                                   `json:"type"`
	Name        string                                                   `json:"name"`
	Description string                                                   `json:"description"`
	Entries     kbapi.SecurityExceptionsAPIExceptionListItemEntryArray   `json:"entries"`
	OsTypes     []string                                                 `json:"os_types,omitempty"`
	Tags        []string                                                 `json:"tags,omitempty"`
	Meta        map[string]any                                           `json:"meta,omitempty"`
	ExpireTime  *string                                                  `json:"expire_time,omitempty"`
	Comments    []ExceptionItemBulkUpdateComment                         `json:"comments,omitempty"`
}

// ExceptionItemBulkUpdateComment is a comment payload for bulk update.
type ExceptionItemBulkUpdateComment struct {
	ID      *string `json:"id,omitempty"`
	Comment string  `json:"comment"`
}

// ExceptionItemBulkUpdateRequest is the request body for bulk update.
type ExceptionItemBulkUpdateRequest struct {
	ListID        string                                `json:"list_id"`
	NamespaceType string                                `json:"namespace_type,omitempty"`
	Items         []ExceptionItemBulkUpdateItemRequest  `json:"items"`
}

// ExceptionItemBulkDeleteRequest is the request body for bulk delete.
type ExceptionItemBulkDeleteRequest struct {
	IDs           []string `json:"ids"`
	NamespaceType string   `json:"namespace_type,omitempty"`
}

// ExceptionItemBulkOperationError is an error returned for a single item in a bulk operation.
type ExceptionItemBulkOperationError struct {
	ItemID *string `json:"item_id,omitempty"`
	ListID *string `json:"list_id,omitempty"`
	Error  struct {
		Message    string `json:"message"`
		StatusCode int    `json:"status_code"`
	} `json:"error"`
}

// ExceptionItemBulkResponse is the response from bulk create or update.
type ExceptionItemBulkResponse struct {
	Items  []kbapi.SecurityExceptionsAPIExceptionListItem `json:"items"`
	Errors []ExceptionItemBulkOperationError              `json:"errors"`
}

// ExceptionItemBulkDeleteResponse is the response from bulk delete.
type ExceptionItemBulkDeleteResponse struct {
	Items  []kbapi.SecurityExceptionsAPIExceptionListItem `json:"items"`
	Errors []ExceptionItemBulkOperationError              `json:"errors"`
}

const (
	bulkCreateExceptionItemsPath = "/api/exception_lists/items/_bulk"
	bulkUpdateExceptionItemsPath = "/api/exception_lists/items/_bulk_update"
	bulkDeleteExceptionItemsPath = "/api/exception_lists/items/_bulk_delete"
)

// BulkCreateExceptionListItems sends a bulk create request for exception list items.
func BulkCreateExceptionListItems(
	ctx context.Context,
	client *Client,
	spaceID string,
	req ExceptionItemBulkCreateRequest,
) (*ExceptionItemBulkResponse, diag.Diagnostics) {
	path := kibanautil.BuildSpaceAwarePath(spaceID, bulkCreateExceptionItemsPath)
	return doExceptionItemBulkRequest[ExceptionItemBulkCreateRequest, ExceptionItemBulkResponse](
		ctx, client, http.MethodPost, path, req,
	)
}

// BulkUpdateExceptionListItems sends a bulk update request for exception list items.
func BulkUpdateExceptionListItems(
	ctx context.Context,
	client *Client,
	spaceID string,
	req ExceptionItemBulkUpdateRequest,
) (*ExceptionItemBulkResponse, diag.Diagnostics) {
	path := kibanautil.BuildSpaceAwarePath(spaceID, bulkUpdateExceptionItemsPath)
	return doExceptionItemBulkRequest[ExceptionItemBulkUpdateRequest, ExceptionItemBulkResponse](
		ctx, client, http.MethodPut, path, req,
	)
}

// BulkDeleteExceptionListItems sends a bulk delete request for exception list items.
func BulkDeleteExceptionListItems(
	ctx context.Context,
	client *Client,
	spaceID string,
	req ExceptionItemBulkDeleteRequest,
) (*ExceptionItemBulkDeleteResponse, diag.Diagnostics) {
	path := kibanautil.BuildSpaceAwarePath(spaceID, bulkDeleteExceptionItemsPath)
	return doExceptionItemBulkRequest[ExceptionItemBulkDeleteRequest, ExceptionItemBulkDeleteResponse](
		ctx, client, http.MethodPost, path, req,
	)
}

// FindExceptionListItemsAllPages reads all exception list items for a list_id by
// paginating through the _find endpoint. Returns the full list.
func FindExceptionListItemsAllPages(
	ctx context.Context,
	client *Client,
	spaceID string,
	listID string,
	namespaceType string,
) ([]kbapi.SecurityExceptionsAPIExceptionListItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	const perPage = 1000

	var all []kbapi.SecurityExceptionsAPIExceptionListItem
	page := 1

	nsType := kbapi.SecurityExceptionsAPIExceptionNamespaceType(namespaceType)
	for {
		p := page
		pp := perPage
		listIDs := []kbapi.SecurityExceptionsAPIExceptionListHumanId{kbapi.SecurityExceptionsAPIExceptionListHumanId(listID)}
		params := &kbapi.FindExceptionListItemsParams{
			ListId:        listIDs,
			NamespaceType: &[]kbapi.SecurityExceptionsAPIExceptionNamespaceType{nsType},
			Page:          &p,
			PerPage:       &pp,
		}

		resp, err := client.API.FindExceptionListItemsWithResponse(
			ctx, params, kibanautil.SpaceAwarePathRequestEditor(spaceID),
		)
		if err != nil {
			diags.AddError("Failed to list exception items", err.Error())
			return nil, diags
		}

		if resp.StatusCode() != http.StatusOK {
			diags.AddError(
				"Failed to list exception items",
				fmt.Sprintf("unexpected status %d: %s", resp.StatusCode(), string(resp.Body)),
			)
			return nil, diags
		}

		if resp.JSON200 == nil {
			break
		}

		all = append(all, resp.JSON200.Data...)

		fetched := page * perPage
		if fetched >= resp.JSON200.Total {
			break
		}
		page++
	}

	return all, diags
}

// doExceptionItemBulkRequest is a generic helper for bulk POST/PUT requests.
func doExceptionItemBulkRequest[Req any, Resp any](
	ctx context.Context,
	client *Client,
	method string,
	path string,
	reqBody Req,
) (*Resp, diag.Diagnostics) {
	var diags diag.Diagnostics

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		diags.AddError("Failed to marshal bulk request", err.Error())
		return nil, diags
	}

	url := strings.TrimSuffix(client.URL, "/") + path
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		diags.AddError("Failed to create bulk request", err.Error())
		return nil, diags
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("kbn-xsrf", "true")
	httpReq.Header.Set("x-elastic-internal-origin", "kibana")
	// Kibana API version header
	httpReq.Header.Set("elastic-api-version", "2023-10-31")

	// wait_for blocks until the write is visible to search, avoiding a race
	// with the envelope's mandatory read-after-write.
	q := httpReq.URL.Query()
	q.Set("refresh", "wait_for")
	httpReq.URL.RawQuery = q.Encode()

	resp, err := client.HTTP.Do(httpReq)
	if err != nil {
		diags.AddError("Failed to execute bulk request", err.Error())
		return nil, diags
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		diags.AddError("Failed to read bulk response body", err.Error())
		return nil, diags
	}

	if resp.StatusCode != http.StatusOK {
		diags.AddError(
			"Bulk request failed",
			fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBytes)),
		)
		return nil, diags
	}

	var result Resp
	if err := json.Unmarshal(respBytes, &result); err != nil {
		diags.AddError("Failed to parse bulk response", err.Error())
		return nil, diags
	}

	return &result, diags
}
