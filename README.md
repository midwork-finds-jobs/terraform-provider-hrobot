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

#### Building the CLI

```bash
# Build the binary
go build -o hrobot cmd/hrobot/main.go

# Or install it to your $GOPATH/bin
go install ./cmd/hrobot
```

#### Usage

[Set your credentials](https://robot.hetzner.com/preferences/index) as environment variables:

```bash
# Set credentials
export HETZNER_ROBOT_USER="your-username"
export HETZNER_ROBOT_PASSWORD="your-password"

# List all servers:
hrobot servers

# Get details for a specific server: 
hrobot server 1234567

# See firewall rules
hrobot firewall get 1234567

# Enable rescue system for server:
hrobot boot rescue enable 1234567

# Boot the system (to enable rescue system)
hrobot reset trigger 1234567 hw
```

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
