terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# Route a failover IP to a specific server
# The failover IP must already exist in your Hetzner account
resource "hrobot_failover" "example" {
  ip               = "123.123.123.123"
  active_server_ip = "12.34.5.67" # Main IP of the destination server
}

# IPv6 failover example
resource "hrobot_failover" "ipv6_example" {
  ip               = "XXXX:YYY:ZZZZ::"
  active_server_ip = "AAAA:BBB:CCC:DDDD::" # Main IPv6 address of the destination server
}

# Import an existing failover IP
# terraform import hrobot_failover.example 123.123.123.123
