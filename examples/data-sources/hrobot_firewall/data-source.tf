terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# lookup firewall configuration for a server
data "hrobot_firewall" "example" {
  server_id = 1234567
}
