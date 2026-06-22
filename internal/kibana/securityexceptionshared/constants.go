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

// Package securityexceptionshared provides shared types, schema definitions,
// and conversion helpers used by both the single-item and bulk exception list
// item resources.
package securityexceptionshared

// Entry type values used in exception item entries.
const (
	EntryTypeMatch    = "match"
	EntryTypeWildcard = "wildcard"
	EntryTypeMatchAny = "match_any"
	EntryTypeList     = "list"
	EntryTypeExists   = "exists"
	EntryTypeNested   = "nested"
)

// Schema attribute name constants shared across exception item schemas.
const (
	AttrType     = "type"
	AttrField    = "field"
	AttrOperator = "operator"
	AttrValue    = "value"
	AttrValues   = "values"
	AttrEntries  = "entries"
)
