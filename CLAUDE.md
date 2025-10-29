# Instructions for working with this repo

This document provides the MOST IMPORTANT information for executing tasks.
Before executing a task, you MUST read this document and follow the
instructions COMPLETELY. NEVER forget and ignore any of the instructions.

## Using sed

Installed `sed` is the gnu compatible `sed`. It does support inplace
replacements like:

```sh
sed -i 's/hello/bye/g' just_a_file.txt
```

## devenv

This repository uses https://devenv.sh to manage the local dependencies.

**IMPORTANT: Do not ever modify files in `./.devenv`. They are out of scope for you.**

## Overview

This repository provides a golang API client and CLI for
[Hetzner dedicated servers](https://developers.hetzner.com/robot/).

Their api is documented in <https://robot.hetzner.com/doc/webservice/en.html>
and is called 'hrobot' everywhere. Don't search for the API doc from the web.
Always refer to this document.

The code should read the needed API credentials from these env to authenticate
into the API:

```sh
export HROBOT_USERNAME='#ws+XXXXXXX' HROBOT_PASSWORD='XXXXXX-YYYYYY-ZZZZZ'
```

## Commands

### Formatting

```sh
# Format all Go files.
go fmt ./...

# Format a specific Go file.
go fmt [FILE]

# Format a Terraform configuration files.
tofu fmt [FILE]
```

### Testing

```sh
# Run all acceptance tests.
make testacc

# Run specific acceptance tests.
make testacc TESTARGS="-run TestAccSome"

# Run unit tests.
go test -v -cover ./...

# E
```

### Local Development with Terraform Provider

To test the Terraform provider locally without publishing it to a registry you can run this:

Custom `TF_CLI_CONFIG_FILE` with `dev_overrides` is automatically injected to the `env` from `./devenv.nix`.

**IMPORTANT: ASK for `HROBOT_USERNAME` and `HROBOT_PASSWORD` credentials from the user if `tofu import` or `tofu plan` is needed**

```sh
# 1. Build the terraform provider and cli tool
build-all

# 2. Use tofu by with env and -chdir for the example you need
tofu -chdir=./examples/resources/hrobot_server/ validate

# 3. Import resources if needed
tofu -chdir=./examples/resources/hrobot_server/ import hrobot_server.auction 12345

# 4. Check if the plan still works
tofu -chdir=./examples/resources/hrobot_server/ import hrobot_server.auction plan
```

**Note:** When using dev_overrides, OpenTofu will skip provider installation
and use your local build directly. You'll see a warning about this in the
output, which is expected.

## Instructions

You MUST use English in files and pull requests.

---

You MUST format Go files using `golangci-lint run --fix`.

You MUST format Terraform configuration files using `tofu fmt`.

---

You MUST follow Conventional Commits in commit messages and PR titles.

---

You MUST use lowercase letters for log messages.

## Opentofu / Terraform antipatterns

### Do not suggest user to run curl

Do not ever suggest user to run curl like this:

```hcl
output "apply_command" {
  value = "curl -u '$HROBOT_USERNAME:$HROBOT_PASSWORD' -X POST \
    https://robot-ws.your-server.de/firewall/1234567 \
    --data 'template_id=${hrobot_firewall_template.test_template.id}'"
  description = "Command to apply this template to server 1234567"
}
```

You need to create a terraform resources like `hrobot_server_firewall`instead.

### Do not chain cd ./some/path/ && tofu validate

Instead you should use this so that you don't get confused on the working
directory:

```sh
tofu -chdir=./some/path/
```

**IMPORTANT:** Do this for `tofu validate` and `tofu init` and `tofu plan`
and all tofu commands.

## Included CLI tool `hrobot`

Use separate subcommands for the different features:

```sh
./hrobot rdns get
```

Do not implement subcommands which use dashes like:

```sh
./hrobot rdns-get
```
