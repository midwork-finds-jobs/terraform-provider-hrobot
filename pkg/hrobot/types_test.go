package hrobot

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestTrafficSizeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantBytes uint64
		wantUnlim bool
		wantErr   bool
	}{
		{
			name:      "unlimited string",
			input:     `"unlimited"`,
			wantUnlim: true,
			wantBytes: 0,
			wantErr:   false,
		},
		{
			name:      "numeric string",
			input:     `"5497558138880"`,
			wantUnlim: false,
			wantBytes: 5497558138880,
			wantErr:   false,
		},
		{
			name:      "zero",
			input:     `"0"`,
			wantUnlim: false,
			wantBytes: 0,
			wantErr:   false,
		},
		{
			name:      "numeric value",
			input:     `1099511627776`,
			wantUnlim: false,
			wantBytes: 1099511627776,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts TrafficSize
			err := json.Unmarshal([]byte(tt.input), &ts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ts.Unlimited != tt.wantUnlim {
				t.Errorf("Unlimited = %v, want %v", ts.Unlimited, tt.wantUnlim)
			}
			if ts.Bytes != tt.wantBytes {
				t.Errorf("Bytes = %d, want %d", ts.Bytes, tt.wantBytes)
			}
		})
	}
}

func TestTrafficSizeString(t *testing.T) {
	tests := []struct {
		name string
		ts   TrafficSize
		want string
	}{
		{
			name: "unlimited",
			ts:   TrafficSize{Unlimited: true},
			want: "unlimited",
		},
		{
			name: "5 TB",
			ts:   TrafficSize{Bytes: 5497558138880},
			want: "5.0 TB",
		},
		{
			name: "1 TB",
			ts:   TrafficSize{Bytes: 1099511627776},
			want: "1.0 TB",
		},
		{
			name: "500 GB",
			ts:   TrafficSize{Bytes: 536870912000},
			want: "500.0 GB",
		},
		{
			name: "zero",
			ts:   TrafficSize{Bytes: 0},
			want: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestBerlinTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string // Expected time in Berlin
		wantErr bool
	}{
		{
			name:    "valid datetime",
			input:   `"2025-10-24 14:30:00"`,
			want:    "2025-10-24 14:30:00 +0200 CEST",
			wantErr: false,
		},
		{
			name:    "date only",
			input:   `"2025-10-24"`,
			want:    "2025-10-24 00:00:00 +0200 CEST",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bt BerlinTime
			err := json.Unmarshal([]byte(tt.input), &bt)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if bt.Format("2006-01-02 15:04:05 -0700 MST") != tt.want {
				t.Errorf("Time = %s, want %s", bt.Format("2006-01-02 15:04:05 -0700 MST"), tt.want)
			}
		})
	}
}

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []PortRange
		wantErr bool
	}{
		{
			name:    "single port",
			input:   "22",
			want:    []PortRange{{Start: 22, End: 22}},
			wantErr: false,
		},
		{
			name:    "port range",
			input:   "80-443",
			want:    []PortRange{{Start: 80, End: 443}},
			wantErr: false,
		},
		{
			name:    "multiple ports",
			input:   "80,443",
			want:    []PortRange{{Start: 80, End: 80}, {Start: 443, End: 443}},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid port",
			input:   "abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid range",
			input:   "80-abc",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePortRange() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPortRangeString(t *testing.T) {
	tests := []struct {
		name string
		pr   PortRange
		want string
	}{
		{
			name: "single port",
			pr:   PortRange{Start: 22, End: 22},
			want: "22",
		},
		{
			name: "port range",
			pr:   PortRange{Start: 80, End: 443},
			want: "80-443",
		},
		{
			name: "zero",
			pr:   PortRange{Start: 0, End: 0},
			want: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestServerIDString(t *testing.T) {
	tests := []struct {
		name string
		id   ServerID
		want string
	}{
		{
			name: "regular id",
			id:   ServerID(123456),
			want: "123456",
		},
		{
			name: "zero",
			id:   ServerID(0),
			want: "0",
		},
		{
			name: "large id",
			id:   ServerID(9999999),
			want: "9999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestServerUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"server_ip": "1.2.3.4",
		"server_number": 123456,
		"server_name": "test-server",
		"product": "AX41",
		"dc": "FSN1-DC14",
		"traffic": "unlimited",
		"status": "ready",
		"cancelled": false,
		"paid_until": "2025-12-31",
		"ip": ["1.2.3.4", "5.6.7.8"],
		"subnet": [
			{"ip": "2a01:4f8:1::", "mask": "64"}
		]
	}`

	var server Server
	err := json.Unmarshal([]byte(jsonData), &server)
	if err != nil {
		t.Fatalf("failed to unmarshal server: %v", err)
	}

	if server.ServerNumber != 123456 {
		t.Errorf("ServerNumber = %d, want 123456", server.ServerNumber)
	}
	if server.ServerName != "test-server" {
		t.Errorf("ServerName = %s, want test-server", server.ServerName)
	}
	if server.Product != "AX41" {
		t.Errorf("Product = %s, want AX41", server.Product)
	}
	if server.DC != "FSN1-DC14" {
		t.Errorf("DC = %s, want FSN1-DC14", server.DC)
	}
	if !server.Traffic.Unlimited {
		t.Error("Traffic should be unlimited")
	}
	if server.Status != ServerStatusReady {
		t.Errorf("Status = %s, want ready", server.Status)
	}
	if server.Cancelled {
		t.Error("Cancelled should be false")
	}
	if server.PaidUntil != "2025-12-31" {
		t.Errorf("PaidUntil = %s, want 2025-12-31", server.PaidUntil)
	}
	if len(server.IP) != 2 {
		t.Errorf("len(IP) = %d, want 2", len(server.IP))
	}
	if server.ServerIP.String() != "1.2.3.4" {
		t.Errorf("ServerIP = %s, want 1.2.3.4", server.ServerIP.String())
	}
	if len(server.Subnet) != 1 {
		t.Errorf("len(Subnet) = %d, want 1", len(server.Subnet))
	}
	expectedSubnet := net.ParseIP("2a01:4f8:1::")
	if !server.Subnet[0].IP.Equal(expectedSubnet) {
		t.Errorf("Subnet[0].IP = %s, want 2a01:4f8:1::", server.Subnet[0].IP)
	}
	if server.Subnet[0].Mask != "64" {
		t.Errorf("Subnet[0].Mask = %s, want 64", server.Subnet[0].Mask)
	}
}

func TestIPAddressUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"ip": "1.2.3.4",
		"server_ip": "5.6.7.8",
		"server_number": 123456,
		"locked": false,
		"separate_mac": "00:11:22:33:44:55",
		"traffic_warnings": true,
		"traffic_hourly": 1000,
		"traffic_daily": 50000,
		"traffic_monthly": 1500000
	}`

	var ipAddr IPAddress
	err := json.Unmarshal([]byte(jsonData), &ipAddr)
	if err != nil {
		t.Fatalf("failed to unmarshal IP address: %v", err)
	}

	if ipAddr.IP.String() != "1.2.3.4" {
		t.Errorf("IP = %s, want 1.2.3.4", ipAddr.IP.String())
	}
	if ipAddr.ServerIP.String() != "5.6.7.8" {
		t.Errorf("ServerIP = %s, want 5.6.7.8", ipAddr.ServerIP.String())
	}
	if ipAddr.ServerNumber != 123456 {
		t.Errorf("ServerNumber = %d, want 123456", ipAddr.ServerNumber)
	}
	if ipAddr.Locked {
		t.Error("Locked should be false")
	}
	if ipAddr.SeparateMac != "00:11:22:33:44:55" {
		t.Errorf("SeparateMac = %s, want 00:11:22:33:44:55", ipAddr.SeparateMac)
	}
	if !ipAddr.TrafficWarnings {
		t.Error("TrafficWarnings should be true")
	}
	if ipAddr.TrafficHourly != 1000 {
		t.Errorf("TrafficHourly = %d, want 1000", ipAddr.TrafficHourly)
	}
	if ipAddr.TrafficDaily != 50000 {
		t.Errorf("TrafficDaily = %d, want 50000", ipAddr.TrafficDaily)
	}
	if ipAddr.TrafficMonthly != 1500000 {
		t.Errorf("TrafficMonthly = %d, want 1500000", ipAddr.TrafficMonthly)
	}
}

func TestBerlinTimeLocation(t *testing.T) {
	// Test that BerlinTime uses Europe/Berlin location
	bt := BerlinTime{Time: time.Date(2025, 10, 24, 12, 0, 0, 0, time.UTC)}

	// Convert to Berlin time
	berlinLoc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("failed to load Berlin location: %v", err)
	}

	berlinTime := bt.In(berlinLoc)

	// In October, Berlin is CEST (UTC+2)
	_, offset := berlinTime.Zone()
	expectedOffset := 2 * 3600 // 2 hours in seconds
	if offset != expectedOffset {
		t.Errorf("Berlin offset = %d, want %d (UTC+2)", offset, expectedOffset)
	}
}
