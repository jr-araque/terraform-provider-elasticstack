variable "list_id" {
  description = "The exception list ID"
  type        = string
}

variable "item_id_1" {
  description = "The first item ID"
  type        = string
}

variable "item_id_2" {
  description = "The second item ID"
  type        = string
}

provider "elasticstack" {
  elasticsearch {}
  kibana {}
}

resource "elasticstack_kibana_security_exception_list" "test" {
  list_id        = var.list_id
  name           = "Test Bulk Exception List"
  description    = "Test exception list for bulk items acceptance tests"
  type           = "detection"
  namespace_type = "single"
}

resource "elasticstack_kibana_security_exception_items" "test" {
  list_id        = elasticstack_kibana_security_exception_list.test.list_id
  namespace_type = "single"

  items {
    item_id     = var.item_id_1
    type        = "simple"
    name        = "Bulk Item 1"
    description = "First bulk exception item"
    entries = [
      {
        type     = "match"
        field    = "process.name"
        operator = "included"
        value    = "test-process-1"
      }
    ]
  }

  items {
    item_id     = var.item_id_2
    type        = "simple"
    name        = "Bulk Item 2"
    description = "Second bulk exception item"
    entries = [
      {
        type     = "match"
        field    = "host.name"
        operator = "included"
        value    = "test-host-2"
      }
    ]
  }
}
