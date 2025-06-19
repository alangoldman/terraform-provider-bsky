package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccAccountPreCheck(t *testing.T) {
	if os.Getenv("BSKY_PDS_ADMIN_PASSWORD") == "" {
		t.Fatal("BSKY_PDS_ADMIN_PASSWORD must be set for acceptance tests")
	}
}

func TestAccAccountResource(t *testing.T) {
	// Use timestamp to ensure unique handles for each test run
	timestamp := time.Now().Unix()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testProviderPreCheck(t)
			testAccAccountPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAccountResourceConfig(fmt.Sprintf("tf-test-%d", timestamp), "test@example.com", "testpass123"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bsky_account.test", "did"),
					resource.TestCheckResourceAttr("bsky_account.test", "handle", fmt.Sprintf("tf-test-%d.test.bsky.social", timestamp)),
					resource.TestCheckResourceAttr("bsky_account.test", "email", "test@example.com"),
				),
			},
			// ImportState testing
			{
				ResourceName: "bsky_account.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["bsky_account.test"].Primary.Attributes["did"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "did",
				// Password and email can't be imported
				ImportStateVerifyIgnore: []string{"password", "email"},
			},
			// Update and Read testing
			{
				Config: testAccAccountResourceConfig(fmt.Sprintf("tf-test-%d-updated", timestamp), "updated@example.com", "newpass123"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bsky_account.test", "did"),
					resource.TestCheckResourceAttr("bsky_account.test", "handle", fmt.Sprintf("tf-test-%d-updated.test.bsky.social", timestamp)),
					resource.TestCheckResourceAttr("bsky_account.test", "email", "updated@example.com"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAccountResourceConfig(handle string, email string, password string) string {
	return fmt.Sprintf(`
resource "bsky_account" "test" {
	handle   = "%[1]s.test.bsky.social"
	email    = %[2]q
	password = %[3]q
}
`, handle, email, password)
}
