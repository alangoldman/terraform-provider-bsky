package test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccListItemResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testProviderPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccListItemResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bsky_list_item.test", "uri"),
					resource.TestCheckResourceAttrSet("bsky_list_item.test", "list_uri"),
					resource.TestCheckResourceAttrSet("bsky_list_item.test", "subject_did"),
					// Verify the item appears in the list data source
					resource.TestCheckResourceAttr("data.bsky_list.with_member", "list_item_count", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("data.bsky_list.with_member", "items.*", map[string]string{
						"did": "did:example:test",
					}),
				),
			},
			// ImportState testing
			{
				ResourceName: "bsky_list_item.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["bsky_list_item.test"].Primary.Attributes["uri"], nil
				},
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccListItemResourceConfig() string {
	return `
resource "bsky_list" "test" {
	name        = "Test List for Items"
	description = "A test list for the list item tests"
	purpose     = "app.bsky.graph.defs#curatelist"
}

resource "bsky_list_item" "test" {
	list_uri    = bsky_list.test.uri
	subject_did = "did:example:test"
}

data "bsky_list" "with_member" {
	uri = bsky_list.test.uri

	depends_on = [bsky_list_item.test]
}`
}
