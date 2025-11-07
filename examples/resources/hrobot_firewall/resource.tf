terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# configure firewall for a server
resource "hrobot_firewall" "example" {
  server_id                  = 1234567
  whitelist_hetzner_services = true

  input_rules = [
    {
      name             = "allow ssh from office"
      ip_version       = "ipv4"
      action           = "accept"
      protocol         = "tcp"
      destination_port = "22"
      source_ips = [
        "203.0.113.10",
        "203.0.113.11"
      ]
    },
    {
      name             = "allow http"
      ip_version       = "ipv4"
      action           = "accept"
      protocol         = "tcp"
      destination_port = "80"
    },
    {
      name             = "allow https"
      ip_version       = "ipv4"
      action           = "accept"
      protocol         = "tcp"
      destination_port = "443"
    },
  ]
}
