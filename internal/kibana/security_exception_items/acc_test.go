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

package securityexceptionitems_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/internal/acctest"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	kibanaoapi "github.com/elastic/terraform-provider-elasticstack/internal/clients/kibanaoapi"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccResourceExceptionItemsBulkCreate tests only the Create + Read lifecycle.
// Use this when _bulk_update and _bulk_delete are not yet available in Kibana —
// destroy falls back to individual item deletes automatically.
func TestAccResourceExceptionItemsBulkCreate(t *testing.T) {
	listID := fmt.Sprintf("test-exception-items-list-%s", uuid.New().String()[:8])
	itemID1 := fmt.Sprintf("bulk-item-1-%s", uuid.New().String()[:8])
	itemID2 := fmt.Sprintf("bulk-item-2-%s", uuid.New().String()[:8])

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceExceptionItemsDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"list_id":   config.StringVariable(listID),
					"item_id_1": config.StringVariable(itemID1),
					"item_id_2": config.StringVariable(itemID2),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("elasticstack_kibana_security_exception_items.test", "id"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "list_id", listID),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "namespace_type", "single"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.#", "2"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.item_id", itemID1),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.name", "Bulk Item 1"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.1.item_id", itemID2),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.1.name", "Bulk Item 2"),
					resource.TestCheckResourceAttrSet("elasticstack_kibana_security_exception_items.test", "items.0.id"),
					resource.TestCheckResourceAttrSet("elasticstack_kibana_security_exception_items.test", "items.0.created_at"),
				),
			},
		},
	})
}

// TestAccResourceExceptionItems tests the full Create/Update/Delete lifecycle.
// Requires _bulk, _bulk_update, and _bulk_delete Kibana endpoints.
func TestAccResourceExceptionItems(t *testing.T) {
	listID := fmt.Sprintf("test-exception-items-list-%s", uuid.New().String()[:8])
	itemID1 := fmt.Sprintf("bulk-item-1-%s", uuid.New().String()[:8])
	itemID2 := fmt.Sprintf("bulk-item-2-%s", uuid.New().String()[:8])
	itemID3 := fmt.Sprintf("bulk-item-3-%s", uuid.New().String()[:8])

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceExceptionItemsDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"list_id":   config.StringVariable(listID),
					"item_id_1": config.StringVariable(itemID1),
					"item_id_2": config.StringVariable(itemID2),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("elasticstack_kibana_security_exception_items.test", "id"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "list_id", listID),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "namespace_type", "single"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.#", "2"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.item_id", itemID1),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.name", "Bulk Item 1"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.1.item_id", itemID2),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.1.name", "Bulk Item 2"),
					resource.TestCheckResourceAttrSet("elasticstack_kibana_security_exception_items.test", "items.0.id"),
					resource.TestCheckResourceAttrSet("elasticstack_kibana_security_exception_items.test", "items.0.created_at"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update_add"),
				ConfigVariables: config.Variables{
					"list_id":   config.StringVariable(listID),
					"item_id_1": config.StringVariable(itemID1),
					"item_id_2": config.StringVariable(itemID2),
					"item_id_3": config.StringVariable(itemID3),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.#", "3"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.item_id", itemID1),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.name", "Bulk Item 1 Updated"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.1.item_id", itemID2),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.2.item_id", itemID3),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.2.name", "Bulk Item 3"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update_remove"),
				ConfigVariables: config.Variables{
					"list_id":   config.StringVariable(listID),
					"item_id_1": config.StringVariable(itemID1),
					"item_id_3": config.StringVariable(itemID3),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.#", "2"),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.0.item_id", itemID1),
					resource.TestCheckResourceAttr("elasticstack_kibana_security_exception_items.test", "items.1.item_id", itemID3),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update_remove"),
				ConfigVariables: config.Variables{
					"list_id":   config.StringVariable(listID),
					"item_id_1": config.StringVariable(itemID1),
					"item_id_3": config.StringVariable(itemID3),
				},
				ResourceName:      "elasticstack_kibana_security_exception_items.test",
				ImportState:       true,
				ImportStateVerify: false, // Items order after import may differ
			},
		},
	})
}

func checkResourceExceptionItemsDestroy(s *terraform.State) error {
	client, err := clients.NewAcceptanceTestingKibanaScopedClient()
	if err != nil {
		return err
	}

	oapiClient := client.GetKibanaOapiClient()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticstack_kibana_security_exception_items" {
			continue
		}

		compID, compDiags := clients.CompositeIDFromStr(rs.Primary.ID)
		if compDiags.HasError() {
			return fmt.Errorf("failed to parse resource ID %s", rs.Primary.ID)
		}

		listID := compID.ResourceID
		spaceID := compID.ClusterID
		nsType := rs.Primary.Attributes["namespace_type"]
		if nsType == "" {
			nsType = "single"
		}

		items, diags := kibanaoapi.FindExceptionListItemsAllPages(
			context.Background(), oapiClient, spaceID, listID, nsType,
		)
		if diags.HasError() {
			continue
		}

		if len(items) > 0 {
			return fmt.Errorf("exception items for list %q still exist in space %q (%d items)", listID, spaceID, len(items))
		}
	}
	return nil
}
