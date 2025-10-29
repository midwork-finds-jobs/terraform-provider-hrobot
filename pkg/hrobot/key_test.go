package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKeyService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key" {
			t.Errorf("Expected path '/key', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got '%s'", r.Method)
		}

		response := []map[string]interface{}{
			{
				"key": map[string]interface{}{
					"name":        "test-key-1",
					"fingerprint": "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07",
					"type":        "ED25519",
					"size":        256,
					"data":        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb",
					"created_at":  "2023-06-10 21:34:12",
				},
			},
			{
				"key": map[string]interface{}{
					"name":        "test-key-2",
					"fingerprint": "a1:b2:c3:d4:e5:f6:07:08:09:10:11:12:13:14:15:16",
					"type":        "RSA",
					"size":        4096,
					"data":        "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC...",
					"created_at":  "2023-07-15 10:20:30",
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	keys, err := client.Key.List(ctx)
	if err != nil {
		t.Fatalf("Key.List returned error: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	if keys[0].Name != "test-key-1" {
		t.Errorf("Expected first key name 'test-key-1', got '%s'", keys[0].Name)
	}

	if keys[0].Fingerprint != "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07" {
		t.Errorf("Expected fingerprint 'd7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07', got '%s'", keys[0].Fingerprint)
	}

	if keys[0].Type != "ED25519" {
		t.Errorf("Expected type 'ED25519', got '%s'", keys[0].Type)
	}

	if keys[0].Size != 256 {
		t.Errorf("Expected size 256, got %d", keys[0].Size)
	}
}

func TestKeyService_Get(t *testing.T) {
	fingerprint := "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/key/" + fingerprint
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"key": map[string]interface{}{
				"name":        "test-key",
				"fingerprint": fingerprint,
				"type":        "ED25519",
				"size":        256,
				"data":        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb",
				"created_at":  "2023-06-10 21:34:12",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	key, err := client.Key.Get(ctx, fingerprint)
	if err != nil {
		t.Fatalf("Key.Get returned error: %v", err)
	}

	if key.Name != "test-key" {
		t.Errorf("Expected key name 'test-key', got '%s'", key.Name)
	}

	if key.Fingerprint != fingerprint {
		t.Errorf("Expected fingerprint '%s', got '%s'", fingerprint, key.Fingerprint)
	}
}

func TestKeyService_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key" {
			t.Errorf("Expected path '/key', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		name := r.FormValue("name")
		data := r.FormValue("data")

		if name != "my-new-key" {
			t.Errorf("Expected name 'my-new-key', got '%s'", name)
		}

		if data != "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb" {
			t.Errorf("Unexpected data value")
		}

		response := map[string]interface{}{
			"key": map[string]interface{}{
				"name":        name,
				"fingerprint": "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07",
				"type":        "ED25519",
				"size":        256,
				"data":        data,
				"created_at":  "2023-06-10 21:34:12",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	key, err := client.Key.Create(ctx, "my-new-key", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb")
	if err != nil {
		t.Fatalf("Key.Create returned error: %v", err)
	}

	if key.Name != "my-new-key" {
		t.Errorf("Expected name 'my-new-key', got '%s'", key.Name)
	}

	if key.Type != "ED25519" {
		t.Errorf("Expected type 'ED25519', got '%s'", key.Type)
	}
}

func TestKeyService_Rename(t *testing.T) {
	fingerprint := "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/key/" + fingerprint
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		name := r.FormValue("name")
		if name != "renamed-key" {
			t.Errorf("Expected name 'renamed-key', got '%s'", name)
		}

		response := map[string]interface{}{
			"key": map[string]interface{}{
				"name":        name,
				"fingerprint": fingerprint,
				"type":        "ED25519",
				"size":        256,
				"data":        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb",
				"created_at":  "2023-06-10 21:34:12",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	key, err := client.Key.Rename(ctx, fingerprint, "renamed-key")
	if err != nil {
		t.Fatalf("Key.Rename returned error: %v", err)
	}

	if key.Name != "renamed-key" {
		t.Errorf("Expected name 'renamed-key', got '%s'", key.Name)
	}
}

func TestKeyService_Delete(t *testing.T) {
	fingerprint := "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/key/" + fingerprint
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Key.Delete(ctx, fingerprint)
	if err != nil {
		t.Fatalf("Key.Delete returned error: %v", err)
	}
}

func TestSSHKey_BerlinTime(t *testing.T) {
	jsonData := `{
		"name": "test-key",
		"fingerprint": "d7:34:1c:8c:4e:20:e0:1f:07:66:45:d9:97:22:ec:07",
		"type": "ED25519",
		"size": 256,
		"data": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEaQde8iCKizUOiXlowY1iEL1yCufgjb3aiatGQNPcHb",
		"created_at": "2023-06-10 21:34:12"
	}`

	var key SSHKey
	err := json.Unmarshal([]byte(jsonData), &key)
	if err != nil {
		t.Fatalf("Failed to unmarshal SSH key: %v", err)
	}

	// The time should be parsed as Berlin time (UTC+2 in summer)
	expectedTime := time.Date(2023, 6, 10, 21, 34, 12, 0, time.FixedZone("Europe/Berlin", 2*60*60))

	if !key.CreatedAt.Equal(expectedTime) {
		t.Errorf("Expected created_at to be %v, got %v", expectedTime, key.CreatedAt.Time)
	}
}
