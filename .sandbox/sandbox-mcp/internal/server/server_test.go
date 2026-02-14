package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/jsonrpc"
)

func newTestServer() *Server {
	return New("/workspace/.sandbox/scripts", "/workspace/.sandbox/tools", "test")
}

func TestInitialize(t *testing.T) {
	srv := newTestServer()
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"clientInfo":{"name":"test"}}`),
	}

	resp := srv.HandleRequest(req)
	if resp == nil {
		t.Fatal("Expected response")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("Expected serverInfo")
	}
	if serverInfo["name"] != "sandbox-mcp" {
		t.Errorf("serverInfo.name = %v, want %q", serverInfo["name"], "sandbox-mcp")
	}
}

func TestToolsListRequiresInit(t *testing.T) {
	srv := newTestServer()
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "tools/list",
	}

	resp := srv.HandleRequest(req)
	if resp.Error == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestToolsListAfterInit(t *testing.T) {
	srv := newTestServer()

	// Initialize first
	initReq := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"clientInfo":{"name":"test"}}`),
	}
	srv.HandleRequest(initReq)

	// Now list tools
	listReq := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(2),
		Method:  "tools/list",
	}
	resp := srv.HandleRequest(listReq)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	tools, ok := result["tools"].([]toolDef)
	if !ok {
		t.Fatal("Expected tools array")
	}
	if len(tools) != 7 {
		t.Errorf("Expected 7 tools, got %d", len(tools))
	}
}

func TestToolsCallListScripts(t *testing.T) {
	srv := newTestServer()

	// Initialize
	srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"clientInfo":{"name":"test"}}`),
	})

	// Call list_scripts
	callReq := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(2),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_scripts","arguments":{"category":"utility"}}`),
	}
	resp := srv.HandleRequest(callReq)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Verify response has content
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	content, ok := result["content"].([]map[string]any)
	if !ok {
		t.Fatal("Expected content array")
	}
	if len(content) == 0 {
		t.Fatal("Expected at least one content block")
	}
	text, ok := content[0]["text"].(string)
	if !ok || text == "" {
		t.Error("Expected non-empty text content")
	}
}

func TestHostOnlyScriptRejection(t *testing.T) {
	srv := newTestServer()

	srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"clientInfo":{"name":"test"}}`),
	})

	callReq := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(2),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"run_script","arguments":{"name":"init-host-env.sh"}}`),
	}
	resp := srv.HandleRequest(callReq)

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError=true for host-only script")
	}
}

func TestUnknownMethod(t *testing.T) {
	srv := newTestServer()
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "unknown/method",
	}
	resp := srv.HandleRequest(req)
	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}
	if resp.Error.Code != jsonrpc.CodeMethodNotFound {
		t.Errorf("Error code = %d, want %d", resp.Error.Code, jsonrpc.CodeMethodNotFound)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	srv := newTestServer()

	// Initialize first
	srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"clientInfo":{"name":"test"}}`),
	})

	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	resp := srv.HandleRequest(req)
	if resp != nil {
		t.Error("Expected nil response for notification")
	}
}

// initServer initializes a test server and returns it ready for tools/call.
func initServer() *Server {
	srv := newTestServer()
	srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"clientInfo":{"name":"test"}}`),
	})
	return srv
}

// callTool sends a tools/call request and returns the response.
func callTool(srv *Server, toolName string, argsJSON string) *jsonrpc.Response {
	params := `{"name":"` + toolName + `"`
	if argsJSON != "" {
		params += `,"arguments":` + argsJSON
	}
	params += "}"
	return srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(2),
		Method:  "tools/call",
		Params:  json.RawMessage(params),
	})
}

func TestToolsCallRequiresInit(t *testing.T) {
	srv := newTestServer() // NOT initialized
	resp := srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_scripts"}`),
	})
	if resp.Error == nil {
		t.Error("Expected error when tools/call before initialize")
	}
	if resp.Error.Code != jsonrpc.CodeInternalError {
		t.Errorf("Error code = %d, want %d", resp.Error.Code, jsonrpc.CodeInternalError)
	}
}

func TestToolsCallGetScriptInfo(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "get_script_info", `{"name":"validate-secrets.sh"}`)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected content array with at least one entry")
	}
	text, _ := content[0]["text"].(string)
	if text == "" {
		t.Error("Expected non-empty text content")
	}
	if _, hasErr := result["isError"]; hasErr {
		t.Error("Unexpected isError in response")
	}
}

func TestToolsCallGetScriptInfoMissingName(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "get_script_info", `{}`)
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError=true for missing name param")
	}
}

func TestToolsCallListTools(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "list_tools", "")
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected content array with at least one entry")
	}
	text, _ := content[0]["text"].(string)
	if text == "" {
		t.Error("Expected non-empty text content")
	}
	// Should contain search-history.go
	if !strings.Contains(text, "search-history.go") {
		t.Error("Expected list_tools to include search-history.go")
	}
}

func TestToolsCallGetToolInfo(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "get_tool_info", `{"name":"search-history.go"}`)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected content array with at least one entry")
	}
	text, _ := content[0]["text"].(string)
	if text == "" {
		t.Error("Expected non-empty text content")
	}
	if _, hasErr := result["isError"]; hasErr {
		t.Error("Unexpected isError in response")
	}
}

func TestToolsCallGetToolInfoMissingName(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "get_tool_info", `{}`)
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError=true for missing name param")
	}
}

func TestToolsCallRunScriptMissingName(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "run_script", `{}`)
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError=true for missing name param")
	}
}

func TestToolsCallRunToolMissingName(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "run_tool", `{}`)
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError=true for missing name param")
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "nonexistent_tool", "")
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError=true for unknown tool")
	}
}

func TestToolsCallInvalidParams(t *testing.T) {
	srv := initServer()
	resp := srv.HandleRequest(&jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      float64(2),
		Method:  "tools/call",
		Params:  json.RawMessage(`not valid json`),
	})
	if resp.Error == nil {
		t.Error("Expected JSON-RPC error for invalid params")
	}
	if resp.Error.Code != jsonrpc.CodeInvalidParams {
		t.Errorf("Error code = %d, want %d", resp.Error.Code, jsonrpc.CodeInvalidParams)
	}
}

func TestToolsCallGetUpdateStatus(t *testing.T) {
	srv := initServer()
	resp := callTool(srv, "get_update_status", "")
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected content array with at least one entry")
	}
	text, _ := content[0]["text"].(string)
	if text == "" {
		t.Error("Expected non-empty text content")
	}
	// Should contain update status header
	if !strings.Contains(text, "Template Update Status") {
		t.Error("Expected response to contain 'Template Update Status'")
	}
}

func TestToolsCallListScriptsFilterCategory(t *testing.T) {
	srv := initServer()

	// Filter by "test" category
	resp := callTool(srv, "list_scripts", `{"category":"test"}`)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected map result")
	}
	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected content array")
	}
	text, _ := content[0]["text"].(string)
	// Test category should not contain utility-only scripts
	if strings.Contains(text, `"category": "utility"`) {
		t.Error("Expected only test category scripts when filtering by 'test'")
	}
}
