# hrobot-go

Go client library and Terraform provider for the Hetzner Robot API.

We absolutely love [Hetzner auction servers](https://www.hetzner.com/sb/) but their terraform support was quite lacking.

This is written with the help of Claude Code but tested locally numerous times.

## Components

The repository contains multiple ways to interact with Hetzner:

- [Terraform provider](https://registry.terraform.io/providers/midwork-finds-jobs/hrobot/latest/docs)
- [OpenTofu provider](https://search.opentofu.org/provider/midwork-finds-jobs/hrobot/latest)
- CLI tool `hrobot`

### Terraform Provider

Add the provider to your project:

```hcl
terraform {
  required_providers {
    hrobot = {
      source = "midwork-finds-jobs/hrobot"
    }
  }
}
```

And check for examples in [terraform registry docs](https://registry.terraform.io/providers/midwork-finds-jobs/hrobot/latest/docs).

### CLI Tool

The internal golang api client is also exposed in separate `hrobot` CLI.

#### Installing the CLI

**Homebrew (macOS and Linux):**

```bash
brew install midwork-finds-jobs/tap/hrobot
```

**Go install:**

```bash
go install github.com/midwork-finds-jobs/terraform-provider-hrobot/cmd/hrobot@latest
```

**Build from source:**

```bash
go build -o hrobot cmd/hrobot/main.go
```

#### Usage

[Set your credentials](https://robot.hetzner.com/preferences/index) as environment variables:

```bash
# Set credentials
export HROBOT_USERNAME='#ws+XXXXXXX'
export HROBOT_PASSWORD='YYYYYY'

# Server Management
hrobot server list                           # List all servers
hrobot server describe 1234567               # Get server details
hrobot server reboot 1234567                 # Reboot server
hrobot server ssh 1234567                    # SSH with auto-firewall config
hrobot server ssh 1234567 --user admin       # SSH as specific user

# Firewall Management
hrobot firewall list-rules 1234567           # Show firewall rules
hrobot firewall allow-ssh 1234567 --my-ip    # Allow SSH from your IP
hrobot firewall allow-https 1234567 --source-ips 1.2.3.4
hrobot firewall allow-mosh 1234567 --my-ip   # Allow MOSH access
hrobot firewall allow-all 1234567 --my-ip    # Allow all traffic (use with caution)
hrobot firewall enable 1234567               # Enable firewall
hrobot firewall disable 1234567              # Disable firewall
hrobot firewall status 1234567               # Show firewall status

# Advanced Firewall Rules
hrobot firewall add-rule 1234567 --direction in --protocol tcp --port 8080 --source-ip 1.2.3.4 --action accept
hrobot firewall delete-rule 1234567 --name "rule-name"

# Rescue System
hrobot server enable-rescue 1234567          # Enable Linux rescue
hrobot server enable-rescue 1234567 --vkvm   # Enable VNC rescue
hrobot server disable-rescue 1234567         # Disable rescue

# Server Auctions
hrobot auction list                          # List auction servers
hrobot auction list --location FSN1          # Filter by location
hrobot auction list --gpu-only               # Show only GPU servers

# SSH Keys
hrobot ssh-key list                          # List SSH keys
hrobot ssh-key create --name "mykey" --data "ssh-rsa ..."

# Reverse DNS
hrobot rdns list                             # List reverse DNS entries
hrobot rdns set 1.2.3.4 hostname.example.com
```

#### CLI Features

##### Intelligent SSH Connection

- Automatically checks if SSH port is accessible
- Auto-configures firewall rules if needed
- Waits for changes to propagate
- Seamlessly opens SSH connection

##### Smart Firewall Management

- Detects pending firewall changes and waits automatically
- Prevents duplicate rules
- Provides helpful error messages for conflicts
- Validates against Hetzner's 10-rule limit

##### Convenience Commands

- `allow-ssh`, `allow-https`, `allow-mosh`, `allow-all` - Quick access rules
- `--my-ip` flag to auto-detect your public IP
- Support for multiple IPs/CIDRs with `--source-ips`

## Development

Install and activate [devenv](https://devenv.sh). There are quite a few hacks needed to build and test terraform plugins locally.

This also ensures that you have proper git hooks in place. See more how we use devenv by looking at `./devenv.nix`.

### Building the Library

```bash
go mod download

# Build terraform provider
go build -v -o terraform-provider-hrobot

# Build 'hrobot' cli
go build -v -o hrobot cmd/hrobot/main.go
```

### Running Tests

The repository includes two types of tests:

#### Unit Tests

Unit tests use mocked HTTP responses and don't require credentials. They run quickly and test the API client logic:

```bash
# Using make (recommended)
make test

# Or using go directly
go test -v -cover ./...
```

#### Acceptance Tests (Integration Tests)

⚠️ **Warning:** Acceptance tests make REAL API calls to Hetzner and costs real money!

```bash
# Set credentials first
export HROBOT_USERNAME='#ws+XXXXXXX'
export HROBOT_PASSWORD='YYYYYY'

# Run acceptance tests
make testacc
```

### Debugging

#### Enable Debug Logging

You can enable verbose HTTP request/response logging for the provider by setting the `HROBOT_DEBUG` environment variable:

```bash
# Enable debug logging
export HROBOT_DEBUG=1

# Run OpenTofu/Terraform with debug logging
tofu apply

# Or for a single command
HROBOT_DEBUG=1 tofu apply
```

When enabled, the provider will output detailed information about:

- HTTP request URLs and methods
- Request headers and body
- HTTP response status codes
- Response headers and body

This is useful for troubleshooting API issues or understanding what requests are being made to the Hetzner Robot API.

**Note:** Debug output may contain sensitive information. Be careful when sharing logs publicly.

## API Documentation

Full Hetzner Robot API documentation: https://robot.hetzner.com/doc/webservice/en.html

## License

Mozilla Public License Version 2.0 - see [LICENSE](./LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Notes

- Firewall updates can take 30-40 seconds to apply
- IPv6 filtering has limitations (see Hetzner documentation)
- ICMPv6 traffic is always allowed
- Default firewall policy is discard (deny)
- Hetzner blocks outgoing traffic from ports 25 and 465 to prevent spam
- Some product servers have setup fees of 79€. Check before buying.
