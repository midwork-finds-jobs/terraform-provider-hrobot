terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# create a vswitch
resource "hrobot_vswitch" "example" {
  name = "my-vswitch"
  vlan = 4000
}
