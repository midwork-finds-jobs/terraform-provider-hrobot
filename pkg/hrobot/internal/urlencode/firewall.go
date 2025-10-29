package urlencode

import (
	"fmt"
	"net/url"
	"strconv"
)

// EncodeFirewallRules encodes firewall rules into Hetzner's hierarchical format
// Example: rules[input][0][name]=rule1&rules[input][0][action]=accept.
// Note: Returns a string instead of url.Values because Hetzner's API expects
// brackets in keys to NOT be URL-encoded.
func EncodeFirewallRules(rules map[string][]map[string]string) string {
	var parts []string

	for direction, ruleList := range rules {
		for i, rule := range ruleList {
			for key, value := range rule {
				// Build hierarchical key: rules[direction][index][field]
				// Encode only the value, not the key (brackets must stay literal)
				hierKey := fmt.Sprintf("rules[%s][%d][%s]", direction, i, key)
				encodedValue := url.QueryEscape(value)
				parts = append(parts, fmt.Sprintf("%s=%s", hierKey, encodedValue))
			}
		}
	}

	return joinParts(parts, "&")
}

// joinParts joins string parts with a separator.
func joinParts(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// FirewallRuleEncoder helps build firewall rules.
type FirewallRuleEncoder struct {
	rules map[string][]map[string]string
}

// NewFirewallRuleEncoder creates a new encoder.
func NewFirewallRuleEncoder() *FirewallRuleEncoder {
	return &FirewallRuleEncoder{
		rules: make(map[string][]map[string]string),
	}
}

// AddInputRule adds an input rule.
func (e *FirewallRuleEncoder) AddInputRule(rule map[string]string) {
	if e.rules["input"] == nil {
		e.rules["input"] = []map[string]string{}
	}
	e.rules["input"] = append(e.rules["input"], rule)
}

// AddOutputRule adds an output rule.
func (e *FirewallRuleEncoder) AddOutputRule(rule map[string]string) {
	if e.rules["output"] == nil {
		e.rules["output"] = []map[string]string{}
	}
	e.rules["output"] = append(e.rules["output"], rule)
}

// Encode returns the encoded form string.
func (e *FirewallRuleEncoder) Encode() string {
	return EncodeFirewallRules(e.rules)
}

// EncodeToValues returns url.Values (deprecated, use Encode or EncodeToString).
func (e *FirewallRuleEncoder) EncodeToValues() url.Values {
	// This is kept for compatibility but shouldn't be used with Hetzner API
	values := url.Values{}
	rulesStr := EncodeFirewallRules(e.rules)
	// Parse the string back to values (not ideal but maintains compatibility)
	parsed, _ := url.ParseQuery(rulesStr)
	for k, v := range parsed {
		for _, val := range v {
			values.Add(k, val)
		}
	}
	return values
}

// EncodeToString returns the complete encoded form string with additional values.
func (e *FirewallRuleEncoder) EncodeToString(additional map[string]string) string {
	var parts []string

	// Add additional values first
	for key, value := range additional {
		encodedValue := url.QueryEscape(value)
		parts = append(parts, fmt.Sprintf("%s=%s", key, encodedValue))
	}

	// Add rules
	rulesStr := e.Encode()
	if rulesStr != "" {
		parts = append(parts, rulesStr)
	}

	return joinParts(parts, "&")
}

// MergeValues merges additional values into the encoder's output (deprecated).
func (e *FirewallRuleEncoder) MergeValues(additional url.Values) url.Values {
	// Convert additional to map
	additionalMap := make(map[string]string)
	for key, values := range additional {
		if len(values) > 0 {
			additionalMap[key] = values[0]
		}
	}

	// Get the string
	str := e.EncodeToString(additionalMap)

	// Parse back to url.Values
	values, _ := url.ParseQuery(str)
	return values
}

// RuleBuilder helps build individual firewall rules.
type RuleBuilder struct {
	data map[string]string
}

// NewRuleBuilder creates a new rule builder.
func NewRuleBuilder() *RuleBuilder {
	return &RuleBuilder{
		data: make(map[string]string),
	}
}

// Name sets the rule name.
func (r *RuleBuilder) Name(name string) *RuleBuilder {
	r.data["name"] = name
	return r
}

// IPVersion sets the IP version.
func (r *RuleBuilder) IPVersion(version string) *RuleBuilder {
	r.data["ip_version"] = version
	return r
}

// Action sets the action (accept/discard).
func (r *RuleBuilder) Action(action string) *RuleBuilder {
	r.data["action"] = action
	return r
}

// Protocol sets the protocol.
func (r *RuleBuilder) Protocol(protocol string) *RuleBuilder {
	r.data["protocol"] = protocol
	return r
}

// SourceIP sets the source IP.
func (r *RuleBuilder) SourceIP(ip string) *RuleBuilder {
	r.data["src_ip"] = ip
	return r
}

// DestIP sets the destination IP.
func (r *RuleBuilder) DestIP(ip string) *RuleBuilder {
	r.data["dst_ip"] = ip
	return r
}

// SourcePort sets the source port.
func (r *RuleBuilder) SourcePort(port interface{}) *RuleBuilder {
	r.data["src_port"] = toString(port)
	return r
}

// DestPort sets the destination port.
func (r *RuleBuilder) DestPort(port interface{}) *RuleBuilder {
	r.data["dst_port"] = toString(port)
	return r
}

// TCPFlags sets TCP flags.
func (r *RuleBuilder) TCPFlags(flags string) *RuleBuilder {
	r.data["tcp_flags"] = flags
	return r
}

// Build returns the rule data.
func (r *RuleBuilder) Build() map[string]string {
	return r.data
}

// toString converts various types to string.
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}
