---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "bsky_account Resource - bsky"
subcategory: ""
description: |-
  Manage Accounts. This resource requires the provider to be configured with the pds_admin_password .
---

# bsky_account (Resource)

Manage Accounts. This resource requires the provider to be configured with the `pds_admin_password `.

## Example Usage

```terraform
provider "bsky" {
  pds_host           = "https://bsky.social"
  handle             = "scoott.blog"
  pds_admin_password = "<PDS admin password>"
}

resource "bsky_account" "test-account" {
  email  = "test@scoott.blog"
  handle = "test.scoott.blog"
  // if account password is not specified when creating a new user, one will be autogenerated
}


// example using a bsky_account to create a Cloudflare DNS TXT record with the DID to validate the handle
provider "cloudflare" {
  api_token = "<cloudflare api token>"
}

resource "cloudflare_dns_record" "test-account-dns-verify" {
  zone_id = "<cloudflare zone id>"
  name    = "_atproto.${element(split(".", bsky_account.test-account.handle), 0)}"
  content = "\"did=${bsky_account.test-account.did}\""
  comment = "Bluesky handle verification record for ${bsky_account.test-account.handle}"
  ttl     = 1 // auto
  type    = "TXT"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `handle` (String) Requested handle for the account

### Optional

- `email` (String) The email of the account
- `password` (String, Sensitive) Set the initial account password on create or update the password for an existing account. If not specified on create, a password will be generated and included in the Terraform output in plaintext.

### Read-Only

- `did` (String) Account's DID.

## Import

Import is supported using the following syntax:

```shell
# Accounts can be imported using the DID
terraform import bsky_account.test-account "did:plc:ewvi7nxzyoun6zhxrhs64oiz"
```
