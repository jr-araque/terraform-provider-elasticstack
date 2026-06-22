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

variable "item_id_3" {
  description = "The third item ID"
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
    name        = "Bulk Item 1 Updated"
    description = "First bulk exception item (updated name)"
    entries = [
      {
        type     = "match"
        field    = "process.name"
        operator = "included"
        value    = "test-process-1-updated"
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

  items {
    item_id     = var.item_id_3
    type        = "simple"
    name        = "Bulk Item 3"
    description = "Third bulk exception item (newly added)"
    entries = [
      {
        type     = "match"
        field    = "user.name"
        operator = "included"
        value    = "test-user-3"
      }
    ]
  }
}
