schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_holder = "HashiCorp, Inc."
  copyright_year = 2021

  header_ignore = [
    # Devenv state directory
    ".devenv/**",
    # Devenv files
    "devenv*",
    # Git directory
    ".git/**",
    # Build artifacts
    "dist/**",
    # Generated documentation
    "docs/**",
    # Changelog directory
    ".changelog/**",
    # Metadata files
    "META.d/**",
    # Example files
    "examples/**",
    # GitHub issue templates
    ".github/ISSUE_TEMPLATE/**",
    # Tool configuration
    ".golangci.yml",
    ".goreleaser.yml",
    ".pre-commit-config.yaml",
  ]
}
