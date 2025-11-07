// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// firewallRuleValidator validates that if protocol is set, ip_version must also be set.
type firewallRuleValidator struct{}

// Description returns a plain text description of the validator's behavior.
func (v firewallRuleValidator) Description(ctx context.Context) string {
	return "ensures that ip_version is specified when protocol is specified"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior.
func (v firewallRuleValidator) MarkdownDescription(ctx context.Context) string {
	return "ensures that `ip_version` is specified when `protocol` is specified"
}

// ValidateList validates the list of firewall rules.
func (v firewallRuleValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	// If the list is null or unknown, skip validation
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()

	// Validate each rule
	for i, ruleAttr := range elements {
		if ruleAttr.IsNull() || ruleAttr.IsUnknown() {
			continue
		}

		ruleObj, ok := ruleAttr.(types.Object)
		if !ok {
			continue
		}

		// Extract protocol and ip_version attributes
		attrs := ruleObj.Attributes()

		protocolAttr, hasProtocol := attrs["protocol"]
		ipVersionAttr, hasIPVersion := attrs["ip_version"]
		nameAttr := attrs["name"]

		// Get rule name for better error messages
		ruleName := ""
		if nameAttr != nil {
			if nameStr, ok := nameAttr.(types.String); ok && !nameStr.IsNull() && !nameStr.IsUnknown() {
				ruleName = nameStr.ValueString()
			}
		}

		// Check if protocol is set but ip_version is not
		protocolSet := hasProtocol && protocolAttr != nil
		if protocolSet {
			if protocolStr, ok := protocolAttr.(types.String); ok {
				if !protocolStr.IsNull() && !protocolStr.IsUnknown() && protocolStr.ValueString() != "" {
					protocol := protocolStr.ValueString()

					// Protocol is set, check if ip_version is set
					ipVersionSet := hasIPVersion && ipVersionAttr != nil
					if !ipVersionSet {
						resp.Diagnostics.Append(createIPVersionError(i, ruleName, protocol, req.Path))
						return
					}
					if ipVersionStr, ok := ipVersionAttr.(types.String); ok {
						if ipVersionStr.IsNull() || ipVersionStr.IsUnknown() || ipVersionStr.ValueString() == "" {
							resp.Diagnostics.Append(createIPVersionError(i, ruleName, protocol, req.Path))
							return
						}

						// Check for ICMPv6 filtering - not supported by Hetzner
						ipVersion := ipVersionStr.ValueString()
						if protocol == "icmp" && ipVersion == "ipv6" {
							resp.Diagnostics.Append(createICMPv6Error(i, ruleName, req.Path))
							return
						}
					}
				}
			}
		}
	}
}

func createIPVersionError(index int, ruleName, protocol string, attrPath path.Path) diag.Diagnostic {
	ruleDesc := fmt.Sprintf("Rule %d", index+1)
	if ruleName != "" {
		ruleDesc = fmt.Sprintf("Rule %d ('%s')", index+1, ruleName)
	}

	return diag.NewAttributeErrorDiagnostic(
		attrPath,
		"Missing ip_version in firewall rule",
		fmt.Sprintf("%s specifies protocol='%s' but is missing ip_version.\n\n"+
			"According to Hetzner API documentation: \"Without specifying the IP version, it is not possible to filter on a specific protocol.\"\n\n"+
			"Please add ip_version='ipv4' or ip_version='ipv6' to this rule.", ruleDesc, protocol),
	)
}

func createICMPv6Error(index int, ruleName string, attrPath path.Path) diag.Diagnostic {
	ruleDesc := fmt.Sprintf("Rule %d", index+1)
	if ruleName != "" {
		ruleDesc = fmt.Sprintf("Rule %d ('%s')", index+1, ruleName)
	}

	return diag.NewAttributeErrorDiagnostic(
		attrPath,
		"ICMPv6 filtering is not supported",
		fmt.Sprintf("%s attempts to filter ICMP traffic on IPv6, which is not supported by Hetzner.\n\n"+
			"According to Hetzner API documentation: \"It is not possible to filter the ICMPv6 protocol. ICMPv6 traffic to and from the server is always allowed.\"\n\n"+
			"Please either:\n"+
			"  • Change ip_version to 'ipv4' to filter ICMPv4, or\n"+
			"  • Remove this rule (ICMPv6 traffic is always allowed)", ruleDesc),
	)
}

// FirewallRuleProtocolValidator returns a validator that checks protocol/ip_version relationship.
func FirewallRuleProtocolValidator() validator.List {
	return firewallRuleValidator{}
}
