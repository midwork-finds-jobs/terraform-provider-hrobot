{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  # This allows to load the terraform config from local directory instead
  env.TF_CLI_CONFIG_FILE = pkgs.writeText ".tofurc" ''
    provider_installation {
      # Use from local directory
      dev_overrides {
        "midwork-finds-jobs/hrobot" = "${config.git.root}"
      }
      # For all other providers, use the registry as normal
      direct {}
    }
  '';

  scripts.build-all.exec = ''
    go build -v -o terraform-provider-hrobot
    go build -v -o hrobot cmd/hrobot/main.go
  '';

  packages = with pkgs; [
    # Needed to write GPG keys to release this into Terraform cloud
    gnupg
  ];

  # Replace sed because Claude can't use the sed on MacOS
  scripts.sed.exec = ''
    ${pkgs.gnused}/bin/sed "$@"
  '';

  # https://devenv.sh/languages/
  languages.go.enable = true;
  languages.opentofu.enable = true;
  languages.terraform.enable = true;

  git-hooks.excludes = [
    ".devenv"
    "vendor"
  ];

  # https://devenv.sh/reference/options/#git-hooks
  git-hooks.hooks = {
    # TF
    terraform-format.enable = true;
    terraform-validate.enable = true;
    # Go files
    golangci-lint = {
      enable = true;
      excludes = [ "tools/.*" ];
    };
    # Nix files
    nixfmt-rfc-style.enable = true;
    # Github Actions
    actionlint.enable = true;
    # Markdown files
    markdownlint = {
      enable = true;
      settings.configuration = {
        # Max 130 line length, except if it's code
        MD013 = {
          line_length = 130;
          code_blocks = false;
        };
        # Allow bare URLs in documentation
        MD034 = false;
      };
    };
    # Try not to leak secrets
    trufflehog.enable = true;
    ripsecrets.enable = true;
  };
}
