# Examples

This directory contains example Terraform configurations for the Hetzner Robot provider.

## Provider Configuration

See [provider/](provider/) for examples of configuring the provider with authentication credentials.

## Resources

- [hrobot_server](resources/hrobot_server/) - manage dedicated servers (auction or product-based)
- [hrobot_ssh_key](resources/hrobot_ssh_key/) - manage ssh public keys
- [hrobot_firewall_template](resources/hrobot_firewall_template/) - create reusable firewall templates
- [hrobot_failover_ip](resources/hrobot_failover_ip/) - configure failover IP-addresses
- [hrobot_firewall](resources/hrobot_firewall/) - configure firewall for a server using the template
- [hrobot_vswitch](resources/hrobot_vswitch/) - manage vswitches for server networking
- [hrobot_rdns](resources/rdns/) - manage Reverse DNS for your server IP-addresses

## Data Sources

- [hrobot_server](data-sources/hrobot_server/) - lookup server information
- [hrobot_auction_servers](data-sources/hrobot_auction_servers/) - list and filter available auction servers
- [hrobot_firewall](data-sources/hrobot_firewall/) - lookup firewall configuration
- [hrobot_firewall_template](data-sources/hrobot_firewall_template/) - lookup firewall templates

## Authentication

All examples require authentication to the Hetzner Robot API. You can provide credentials in two ways:

```bash
export HROBOT_USERNAME='#ws+XXXXXXX'
export HROBOT_PASSWORD='XXXXXX-YYYYYY-ZZZZZ'
```

```hcl
# This is not recommended in production
provider "hrobot" {
  username = "#ws+XXXXXXX"
  password = "XXXXXX-YYYYYY-ZZZZZ"
}
```

## Running Examples

1. Navigate to the example directory
2. Initialize Terraform: `terraform init`
3. Plan changes: `terraform plan`
4. Apply changes: `terraform apply`

## Note

These examples use placeholder values (like server IDs, fingerprints, etc.). Replace them with your actual values before applying.

## Documentation Generation

The terraform-plugin-docs tool uses these files to generate documentation:

- **provider/provider.tf**
- **data-sources/`full data source name`/data-source.tf**
- **resources/`full resource name`/resource.tf**
