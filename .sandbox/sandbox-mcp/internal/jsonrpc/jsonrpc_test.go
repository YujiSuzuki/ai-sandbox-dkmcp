package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestRequestUnmarshal(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	var req Request
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if req.Method != "tools/list" {
		t.Errorf("Method = %q, want %q", req.Method, "tools/list")
	}
	if req.IsNotification() {
		t.Error("Expected non-notification request")
	}
}

func TestNotificationDetection(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	var req Request
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if !req.IsNotification() {
		t.Error("Expected notification (no id)")
	}
}

func TestNewResponse(t *testing.T) {
	resp := NewResponse(1, map[string]string{"key": "value"})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if parsed.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", parsed.JSONRPC, "2.0")
	}
	if parsed.Error != nil {
		t.Error("Expected no error")
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(1, CodeMethodNotFound, "Method not found")
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if parsed.Error == nil {
		t.Fatal("Expected error")
	}
	if parsed.Error.Code != CodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", parsed.Error.Code, CodeMethodNotFound)
	}
}
