terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# create a firewall template
resource "hrobot_firewall_template" "example" {
  name                       = "web-server-template"
  whitelist_hetzner_services = true

  input_rules = [
    {
      name       = "allow ssh"
      ip_version = "ipv4"
      action     = "accept"
      protocol   = "tcp"
      dest_port  = "22"
    },
    {
      name       = "allow http"
      ip_version = "ipv4"
      action     = "accept"
      protocol   = "tcp"
      dest_port  = "80"
    },
    {
      name       = "allow https"
      ip_version = "ipv4"
      action     = "accept"
      protocol   = "tcp"
      dest_port  = "443"
    },
  ]
}
