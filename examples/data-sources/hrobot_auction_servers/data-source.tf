terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

provider "hrobot" {}

# list all available auction servers
data "hrobot_auction_servers" "available" {}

# filter auction servers by criteria
data "hrobot_auction_servers" "filtered" {
  filters = {
    datacenter = ["FSN1-DC14", "NBG1-DC3"]
    min_ram    = 32768 # minimum 32GB RAM
    min_hdd    = 2000  # minimum 2TB HDD
    max_price  = 50.00 # maximum €50/month
  }
}
