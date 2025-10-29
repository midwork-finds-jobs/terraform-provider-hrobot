# hrobot-go

Go client library and Terraform provider for the Hetzner Robot API.

We absolutely love [Hetzner auction servers](https://www.hetzner.com/sb/) but their terraform support was quite lacking.

This is written with the help of Claude Code but tested locally on numerous times.

## Components

The repository contains 2 ways to interact with Hetzner:

- CLI tool `hrobot`
- Terraform / OpenTofu provider

### CLI Tool

A command-line interface is available in `cmd/hrobot/` for quick API interactions.

#### Building the CLI

```bash
# Build the binary
go build -o hrobot cmd/hrobot/main.go

# Or install it to your $GOPATH/bin
go install ./cmd/hrobot
```

#### Usage

Set your credentials as environment variables:

```bash
export HETZNER_ROBOT_USER="your-username"
export HETZNER_ROBOT_PASSWORD="your-password"
```

List all servers:

```bash
./hrobot servers
```

Get details for a specific server:

```bash
./hrobot server 1234567
```

### Terraform Provider

See all examples in [./examples/README.md](./examples/README.md).

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

**Current Coverage:** 61.6% for `pkg/hrobot/`

#### Acceptance Tests (Integration Tests)

⚠️ **Warning:** Acceptance tests make REAL API calls to Hetzner and may create/modify/delete resources!

```bash
# Set credentials first
export HROBOT_USERNAME='#ws+XXXXXXX'
export HROBOT_PASSWORD='XXXXXX-YYYYYY-ZZZZZ'

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
- Some product servers have setup fees of 79€. Check before buying.
