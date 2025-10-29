# Changelog

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
