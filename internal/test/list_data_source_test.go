package test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccListDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testProviderPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
			// Create the list first
			{
				Config: testAccListDataSourceBaseConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bsky_list.test", "uri"),
				),
			},
			// Now read it with the data source and verify all attributes
			{
				Config: testAccListDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check base attributes
					resource.TestCheckResourceAttr("data.bsky_list.test", "name", "Test List for Data Source"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "description", "A test list for the data source tests"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "purpose", "app.bsky.graph.defs#curatelist"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "uri"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "cid"),
					// Check optional attributes
					resource.TestCheckResourceAttr("data.bsky_list.test", "avatar", ""),
					resource.TestCheckResourceAttr("data.bsky_list.test", "list_item_count", "0"),
					// Check empty items array
					resource.TestCheckResourceAttr("data.bsky_list.test", "items.#", "0"),
				),
			},
			// Add an item to the list and verify it appears in the data source
			{
				Config: testAccListDataSourceWithItemConfig(strings.Replace(os.Getenv("BSKY_PDS_HOST"), "https://", "", 1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check base attributes still match
					resource.TestCheckResourceAttr("data.bsky_list.test", "name", "Test List for Data Source"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "description", "A test list for the data source tests"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "purpose", "app.bsky.graph.defs#curatelist"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "uri"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "cid"),
					// Check the item count and list items
					resource.TestCheckResourceAttr("data.bsky_list.test", "list_item_count", "1"),
					resource.TestCheckResourceAttr("data.bsky_list.test", "items.#", "1"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "items.0.did"),
					resource.TestCheckResourceAttrSet("data.bsky_list.test", "items.0.uri"),
				),
			},
		},
	})
}

func testAccListDataSourceBaseConfig() string {
	return `
resource "bsky_list" "test" {
	name        = "Test List for Data Source"
	description = "A test list for the data source tests"
	purpose     = "app.bsky.graph.defs#curatelist"
}`
}

func testAccListDataSourceConfig() string {
	return testAccListDataSourceBaseConfig() + `

data "bsky_list" "test" {
	uri = bsky_list.test.uri
}`
}

func testAccListDataSourceWithItemConfig(host string) string {
	return testAccListDataSourceBaseConfig() + fmt.Sprintf(`

resource "bsky_account" "test" {
	handle = "testusr.%[1]s"
	email = "test@example.com"
}

resource "bsky_list_item" "test" {
	list_uri = bsky_list.test.uri
	subject_did = bsky_account.test.did
}

data "bsky_list" "test" {
	uri = bsky_list.test.uri
}`, host)
}
