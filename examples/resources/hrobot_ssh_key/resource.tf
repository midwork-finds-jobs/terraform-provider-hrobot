terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# create an ssh key
resource "hrobot_ssh_key" "example" {
  name       = "my-ssh-key"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb user@example.com"
}
