terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# purchase an auction server
resource "hrobot_server" "auction" {
  server_type = "auction"
  server_id   = 1234567 # the auction server number
  server_name = "my-auction-server"

  # authentication: use either authorized_keys or password
  authorized_keys = [
    "fb:9e:3d:ce:ca:a4:0b:35:4e:4c:87:67:bb:78:50:4b"
  ]

  # optional: specify image/distribution
  image = "Ubuntu 22.04 LTS"

  public_net {
    ipv4_enabled = true
  }
}

# order a product server
resource "hrobot_server" "product" {
  server_type = "AX41-NVMe"
  server_name = "my-product-server"
  datacenter  = "HEL1" # required for product servers

  authorized_keys = [
    "fb:9e:3d:ce:ca:a4:0b:35:4e:4c:87:67:bb:78:50:4b"
  ]

  image = "Ubuntu 22.04 LTS"

  public_net {
    ipv4_enabled = true
  }
}
