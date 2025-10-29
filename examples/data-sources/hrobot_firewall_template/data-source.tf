terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# lookup a firewall template by id
data "hrobot_firewall_template" "example" {
  template_id = "12345"
}
