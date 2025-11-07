# Changelog

## 0.2.0 (2024-11-07)

BREAKING CHANGES:

- **firewall**: Renamed `source_ip` to `source_ips` (now requires list syntax)
- **firewall**: Renamed `destination_ip` to `destination_ips` (now requires list syntax)
- **firewall**: Changed `filter_ipv6` default from `false` to `true` for better security

FEATURES:

- **firewall**: Add automatic IP array expansion - rules with multiple source/destination IPs are now automatically
  expanded into individual firewall rules
- **firewall**: Add validation to ensure expanded rules don't exceed Hetzner's 10-rule limit per direction
- **firewall**: Automatically append `/32` CIDR notation to IPv4 addresses without it
- **server**: Fix import to properly populate `public_net` block, resolving output evaluation errors

IMPROVEMENTS:

- **docs**: Update all examples to demonstrate array syntax for firewall IPs
- **docs**: Clarify that IP attributes are lists, not single values

## 0.1.2

Rename `hrobot_failover` to `hrobot_failover_ip`.

## 0.1.1

Remove the endpoint configuration from the provider to make it more simple.

## 0.1.0

FEATURES: add complete terraform provider and cli for hetzner robot api

Resources:

- hrobot_server: manage dedicated servers (auction/product)
- hrobot_ssh_key: manage ssh public keys
- hrobot_firewall: configure server firewall rules
- hrobot_firewall_template: create reusable firewall templates
- hrobot_firewall_association: apply templates to servers
- hrobot_vswitch: manage virtual switches
- hrobot_rdns: manage reverse dns records
- hrobot_failover_ip: manage failover ips

Data sources:

- hrobot_server: lookup server information
- hrobot_auction_servers: list/filter auction servers
- hrobot_firewall: lookup firewall configuration
- hrobot_firewall_template: lookup firewall templates
