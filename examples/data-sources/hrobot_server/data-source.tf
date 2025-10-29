terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# lookup server information by server id
data "hrobot_server" "example" {
  server_id = 1234567
}
