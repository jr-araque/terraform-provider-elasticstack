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

package kibanautil

import (
	"context"
	"net/http"
)

// WithRefreshWaitFor is a RequestEditorFn that appends refresh=wait_for to
// the request URL. This causes Elasticsearch to block the write response until
// the written documents are visible to search, without forcing a synchronous
// index flush. It is the correct value for write operations that are
// immediately followed by a read (e.g. the provider's mandatory read-after-write).
//
// Use refresh=false (the ES default) only for fire-and-forget writes where
// subsequent reads do not need to observe the written documents.
func WithRefreshWaitFor(_ context.Context, req *http.Request) error {
	q := req.URL.Query()
	q.Set("refresh", "wait_for")
	req.URL.RawQuery = q.Encode()
	return nil
}
