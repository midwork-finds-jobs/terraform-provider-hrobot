terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}

# configure the hetzner robot provider
provider "hrobot" {
  # authentication credentials can be provided via:
  # - environment variables: HROBOT_USERNAME and HROBOT_PASSWORD
  # - or explicitly in the provider configuration (not recommended for production)

  # username = "#ws+XXXXXXX"
  # password = "XXXXXX-YYYYYY-ZZZZZ"

  # optional: override the default api endpoint
  # endpoint = "https://robot-ws.your-server.de"
}
