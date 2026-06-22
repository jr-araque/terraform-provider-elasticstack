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

import "testing"

func TestParseImportID(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantID     string
		wantNsType string
	}{
		{
			name:       "default namespace — no suffix",
			raw:        "default/my-list",
			wantID:     "default/my-list",
			wantNsType: "",
		},
		{
			name:       "agnostic suffix",
			raw:        "default/my-list/agnostic",
			wantID:     "default/my-list",
			wantNsType: "agnostic",
		},
		{
			name:       "explicit single suffix",
			raw:        "default/my-list/single",
			wantID:     "default/my-list",
			wantNsType: "single",
		},
		{
			name:       "list_id with slashes and agnostic suffix",
			raw:        "default/my/nested/list/agnostic",
			wantID:     "default/my/nested/list",
			wantNsType: "agnostic",
		},
		{
			name:       "list_id with slashes and no suffix",
			raw:        "default/my/nested/list",
			wantID:     "default/my/nested/list",
			wantNsType: "",
		},
		{
			name:       "non-default space with agnostic suffix",
			raw:        "prod-space/my-list/agnostic",
			wantID:     "prod-space/my-list",
			wantNsType: "agnostic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotNsType := parseImportID(tc.raw)
			if gotID != tc.wantID {
				t.Errorf("id: got %q, want %q", gotID, tc.wantID)
			}
			if gotNsType != tc.wantNsType {
				t.Errorf("nsType: got %q, want %q", gotNsType, tc.wantNsType)
			}
		})
	}
}
