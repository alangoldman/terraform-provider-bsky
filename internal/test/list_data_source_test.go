package test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccListDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testProviderPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and check a list first
			{
				Config: testAccListDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bsky_list.test", "name", "Test List for Data Source"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "description", "A test list for the data source tests"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "purpose", "app.bsky.graph.defs#curatelist"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "uri"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "cid"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "list_item_count", "0"),
				),
			},
		},
	})
}

func testAccListDataSourceConfig() string {
	return `
resource "bsky_list" "test" {
	name        = "Test List for Data Source"
	description = "A test list for the data source tests"
	purpose     = "app.bsky.graph.defs#curatelist"
}

data "bsky_list" "test" {
	uri = bsky_list.test.uri
}`
}
