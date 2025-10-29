package hrobot

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// IPVersion represents the IP protocol version.
type IPVersion string

const (
	IPv4 IPVersion = "ipv4"
	IPv6 IPVersion = "ipv6"
)

// Action represents a firewall rule action.
type Action string

const (
	ActionAccept  Action = "accept"
	ActionDiscard Action = "discard"
)

// Protocol represents network protocol.
type Protocol string

const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolUDP  Protocol = "udp"
	ProtocolICMP Protocol = "icmp"
	ProtocolESP  Protocol = "esp"
	ProtocolGRE  Protocol = "gre"
)

// ServerID represents a server identifier.
type ServerID int

func (s ServerID) String() string {
	return strconv.Itoa(int(s))
}

// IPAddress represents an IP address with additional metadata.
type IPAddress struct {
	IP              net.IP `json:"ip"`
	ServerIP        net.IP `json:"server_ip"`
	ServerNumber    int    `json:"server_number"`
	Locked          bool   `json:"locked"`
	SeparateMac     string `json:"separate_mac,omitempty"`
	TrafficWarnings bool   `json:"traffic_warnings"`
	TrafficHourly   int    `json:"traffic_hourly"`
	TrafficDaily    int    `json:"traffic_daily"`
	TrafficMonthly  int    `json:"traffic_monthly"`
}

// ServerStatus represents the status of a server.
type ServerStatus string

const (
	ServerStatusReady     ServerStatus = "ready"
	ServerStatusInProcess ServerStatus = "in process"
	ServerStatusCancelled ServerStatus = "cancelled"
)

// ResetType represents different reset types.
type ResetType string

const (
	ResetTypeSoftware  ResetType = "sw"
	ResetTypeHardware  ResetType = "hw"
	ResetTypePower     ResetType = "power"
	ResetTypePowerLong ResetType = "power_long"
	ResetTypeManual    ResetType = "man"
)

// TrafficSize represents traffic with support for "unlimited".
type TrafficSize struct {
	Unlimited bool
	Bytes     uint64
}

// UnmarshalJSON handles "unlimited" string and numeric values.
func (t *TrafficSize) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "unlimited" {
			t.Unlimited = true
			return nil
		}
		// Try parsing as numeric string
		bytes, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid traffic size: %s", str)
		}
		t.Bytes = bytes
		return nil
	}

	// Try as number
	var num uint64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	t.Bytes = num
	return nil
}

func (t TrafficSize) MarshalJSON() ([]byte, error) {
	if t.Unlimited {
		return json.Marshal("unlimited")
	}
	return json.Marshal(t.Bytes)
}

func (t TrafficSize) String() string {
	if t.Unlimited {
		return "unlimited"
	}
	return formatBytes(t.Bytes)
}

// formatBytes converts bytes to human-readable format.
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// BerlinTime represents a timestamp in Europe/Berlin timezone.
type BerlinTime struct {
	time.Time
}

var berlinLocation *time.Location

func init() {
	var err error
	berlinLocation, err = time.LoadLocation("Europe/Berlin")
	if err != nil {
		// Fallback to UTC+1
		berlinLocation = time.FixedZone("CET", 3600)
	}
}

// UnmarshalJSON parses timestamp and converts to Berlin time.
func (bt *BerlinTime) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Parse various timestamp formats from Hetzner API
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.ParseInLocation(format, str, berlinLocation)
		if err == nil {
			bt.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse timestamp: %s", str)
}

func (bt BerlinTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(bt.In(berlinLocation).Format("2006-01-02 15:04:05"))
}

// StringFloat represents a float that is encoded as a string in JSON.
type StringFloat float64

// UnmarshalJSON handles string-encoded floats.
func (sf *StringFloat) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		// Try as number directly
		var f float64
		if err := json.Unmarshal(data, &f); err != nil {
			return err
		}
		*sf = StringFloat(f)
		return nil
	}

	// Parse string as float
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("invalid float string: %s", str)
	}
	*sf = StringFloat(f)
	return nil
}

func (sf StringFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%.4f", float64(sf)))
}

func (sf StringFloat) Float64() float64 {
	return float64(sf)
}

// PortRange represents a port or range of ports.
type PortRange struct {
	Start uint16
	End   uint16
}

// ParsePortRange parses port specifications like "80", "80-443", "80,443".
func ParsePortRange(s string) ([]PortRange, error) {
	if s == "" {
		return nil, nil
	}

	// Handle comma-separated ports
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		ranges := make([]PortRange, 0, len(parts))
		for _, part := range parts {
			rs, err := ParsePortRange(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			ranges = append(ranges, rs...)
		}
		return ranges, nil
	}

	// Handle range
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		start, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port start: %s", parts[0])
		}
		end, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port end: %s", parts[1])
		}
		return []PortRange{{Start: uint16(start), End: uint16(end)}}, nil
	}

	// Single port
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", s)
	}
	return []PortRange{{Start: uint16(port), End: uint16(port)}}, nil
}

func (p PortRange) String() string {
	if p.Start == p.End {
		return strconv.Itoa(int(p.Start))
	}
	return fmt.Sprintf("%d-%d", p.Start, p.End)
}

// Server represents a Hetzner dedicated server.
type Server struct {
	ServerIP     net.IP       `json:"server_ip"`
	ServerNumber int          `json:"server_number"`
	ServerName   string       `json:"server_name"`
	Product      string       `json:"product"`
	DC           string       `json:"dc"`
	Traffic      TrafficSize  `json:"traffic"`
	Status       ServerStatus `json:"status"`
	Cancelled    bool         `json:"cancelled"`
	PaidUntil    string       `json:"paid_until"`
	IP           []net.IP     `json:"ip"`
	Subnet       []Subnet     `json:"subnet"`
}

// Subnet represents a network subnet.
type Subnet struct {
	IP   net.IP `json:"ip"`
	Mask string `json:"mask"`
}

// Reset represents a server reset configuration.
type Reset struct {
	ServerIP        net.IP      `json:"server_ip"`
	ServerIPv6Net   string      `json:"server_ipv6_net,omitempty"`
	ServerNumber    int         `json:"server_number"`
	Type            []ResetType `json:"type"`
	OperatingStatus string      `json:"operating_status,omitempty"`
}
