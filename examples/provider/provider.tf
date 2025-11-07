terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

# Credentials: https://robot.hetzner.com/preferences/index
# NOTE: To purchase servers you need to opt-in:
# Ordering -> ordering over the webservice -> Agree

provider "hrobot" {
  # Use $HROBOT_USERNAME and $HROBOT_PASSWORD env

  # OR:

  # username = "#ws+XXXXXXX"
  # password = "YYYYYYYYYYY"
}
resource "hrobot_ssh_key" "example" {
  name       = "example-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "hrobot_server" "auction" {
  server_type = "auction"
  server_id   = 12345789 # <-- Replace this with the auction server you want
  server_name = "my-auction-server"
  image       = "Rescue system"

  public_net {
    ipv4_enabled = true
  }

  authorized_keys = [hrobot_ssh_key.example.fingerprint]
}

variable "my_ip_address" {
  type        = string
  description = "Allowed IP for SSH ports"
  default     = "127.0.0.1/32"
}

resource "hrobot_firewall_template" "ssh_access" {
  name = "Allow SSH"

  input_rules = [
    {
      name             = "Allow SSH from my IP"
      ip_version       = "ipv4"
      action           = "accept"
      protocol         = "tcp"
      destination_port = "22"
      source_ips       = [var.my_ip_address]
    },
    {
      name             = "Allow establishing TCP"
      ip_version       = "ipv4"
      action           = "accept"
      protocol         = "tcp"
      destination_port = "32768-65535"
      tcp_flags        = "ack"
    },
    {
      name       = "Drop all other input"
      ip_version = "ipv4"
      action     = "discard"
    }
  ]

  output_rules = [
    {
      name       = "Allow all output"
      ip_version = "ipv4"
      action     = "accept"
    }
  ]
}

resource "hrobot_firewall" "server_firewall" {
  server_id   = hrobot_server.auction.server_id
  template_id = hrobot_firewall_template.ssh_access.id
}
