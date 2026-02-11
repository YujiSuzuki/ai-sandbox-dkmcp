package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/executor"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/jsonrpc"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/scriptparser"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/toolparser"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/updatecheck"
)

// Tool definitions for MCP tools/list response.
type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func (s *Server) toolDefinitions() []toolDef {
	return []toolDef{
		{
			Name:        "list_scripts",
			Description: "List available scripts in .sandbox/scripts/ with descriptions and execution environment info",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"description": "Filter by category: utility, test, or all (default: all)",
						"default":     "all",
					},
				},
			},
		},
		{
			Name:        "get_script_info",
			Description: "Get detailed information about a specific script including usage, options, and execution environment",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Script filename (e.g. validate-secrets.sh)",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "run_script",
			Description: "Execute a script in the container. Host-only scripts will be rejected with guidance on how to run them on the host OS",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Script filename (e.g. validate-secrets.sh)",
					},
					"args": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Arguments to pass to the script",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "list_tools",
			Description: "List available tools in .sandbox/tools/",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_tool_info",
			Description: "Get detailed information about a specific tool",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Tool filename (e.g. search-history.go)",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "run_tool",
			Description: "Execute a tool (e.g. go run .sandbox/tools/search-history.go)",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Tool filename (e.g. search-history.go)",
					},
					"args": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Arguments to pass to the tool",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "get_update_status",
			Description: "Check if template updates are available by reading the update check state file",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (s *Server) handleToolsList(req *jsonrpc.Request) *jsonrpc.Response {
	result := map[string]any{
		"tools": s.toolDefinitions(),
	}
	return jsonrpc.NewResponse(req.ID, result)
}

func (s *Server) handleToolsCall(req *jsonrpc.Request) *jsonrpc.Response {
	var params toolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(req.ID, jsonrpc.CodeInvalidParams, "Invalid params")
	}

	switch params.Name {
	case "list_scripts":
		return s.handleListScripts(req.ID, params.Arguments)
	case "get_script_info":
		return s.handleGetScriptInfo(req.ID, params.Arguments)
	case "run_script":
		return s.handleRunScript(req.ID, params.Arguments)
	case "list_tools":
		return s.handleListTools(req.ID)
	case "get_tool_info":
		return s.handleGetToolInfo(req.ID, params.Arguments)
	case "run_tool":
		return s.handleRunTool(req.ID, params.Arguments)
	case "get_update_status":
		return s.handleGetUpdateStatus(req.ID)
	default:
		return jsonrpc.NewResponse(req.ID, errorContent(fmt.Sprintf("Unknown tool: %s", params.Name)))
	}
}

// --- Script handlers ---

func (s *Server) handleListScripts(id any, args json.RawMessage) *jsonrpc.Response {
	var params struct {
		Category string `json:"category"`
	}
	if args != nil {
		if err := json.Unmarshal(args, &params); err != nil {
			return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Invalid arguments: %v", err)))
		}
	}
	if params.Category == "" {
		params.Category = "all"
	}

	scripts, err := scriptparser.ListScripts(s.scriptsDir)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Failed to list scripts: %v", err)))
	}

	// Filter by category
	if params.Category != "all" {
		var filtered []scriptparser.ScriptInfo
		for _, s := range scripts {
			if s.Category == params.Category {
				filtered = append(filtered, s)
			}
		}
		scripts = filtered
	}

	data, _ := json.MarshalIndent(scripts, "", "  ")
	return jsonrpc.NewResponse(id, textContent(string(data)))
}

func (s *Server) handleGetScriptInfo(id any, args json.RawMessage) *jsonrpc.Response {
	var params struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &params); err != nil || params.Name == "" {
		return jsonrpc.NewResponse(id, errorContent("Missing required parameter: name"))
	}

	info, err := scriptparser.GetDetailedInfo(s.scriptsDir, params.Name)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Failed to get script info: %v", err)))
	}

	data, _ := json.MarshalIndent(info, "", "  ")
	return jsonrpc.NewResponse(id, textContent(string(data)))
}

func (s *Server) handleRunScript(id any, args json.RawMessage) *jsonrpc.Response {
	var params struct {
		Name string   `json:"name"`
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(args, &params); err != nil || params.Name == "" {
		return jsonrpc.NewResponse(id, errorContent("Missing required parameter: name"))
	}

	// Check if host-only
	if scriptparser.IsHostOnly(params.Name) {
		msg := fmt.Sprintf(
			"This script (%s) must be run on the host OS, not inside the AI Sandbox.\n\n"+
				"To run it on your host machine:\n"+
				"  .sandbox/scripts/%s %s\n\n"+
				"I cannot execute host-only scripts because the AI Sandbox does not have Docker socket access.",
			params.Name, params.Name, strings.Join(params.Args, " "))
		return jsonrpc.NewResponse(id, errorContent(msg))
	}

	result, err := executor.RunScript(s.scriptsDir, params.Name, params.Args)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Execution failed: %v", err)))
	}

	return jsonrpc.NewResponse(id, textContent(result.String()))
}

// --- Tool handlers ---

func (s *Server) handleListTools(id any) *jsonrpc.Response {
	tools, err := toolparser.ListTools(s.toolsDir)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Failed to list tools: %v", err)))
	}

	data, _ := json.MarshalIndent(tools, "", "  ")
	return jsonrpc.NewResponse(id, textContent(string(data)))
}

func (s *Server) handleGetToolInfo(id any, args json.RawMessage) *jsonrpc.Response {
	var params struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &params); err != nil || params.Name == "" {
		return jsonrpc.NewResponse(id, errorContent("Missing required parameter: name"))
	}

	info, err := toolparser.GetDetailedInfo(s.toolsDir, params.Name)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Failed to get tool info: %v", err)))
	}

	data, _ := json.MarshalIndent(info, "", "  ")
	return jsonrpc.NewResponse(id, textContent(string(data)))
}

func (s *Server) handleRunTool(id any, args json.RawMessage) *jsonrpc.Response {
	var params struct {
		Name string   `json:"name"`
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(args, &params); err != nil || params.Name == "" {
		return jsonrpc.NewResponse(id, errorContent("Missing required parameter: name"))
	}

	result, err := executor.RunTool(s.toolsDir, params.Name, params.Args)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Execution failed: %v", err)))
	}

	return jsonrpc.NewResponse(id, textContent(result.String()))
}

// --- Update check handlers ---

func (s *Server) handleGetUpdateStatus(id any) *jsonrpc.Response {
	// Default paths
	stateFile := "/workspace/.sandbox/.state/update-check"
	configFile := "/workspace/.sandbox/config/template-source.conf"

	// Get update status
	status, err := updatecheck.GetUpdateStatus(stateFile, configFile)
	if err != nil {
		return jsonrpc.NewResponse(id, errorContent(fmt.Sprintf("Failed to get update status: %v", err)))
	}

	// Format response
	var output strings.Builder
	output.WriteString("📦 Template Update Status\n\n")

	if status.LatestVersion != "" {
		output.WriteString(fmt.Sprintf("Latest version: %s\n", status.LatestVersion))
	} else {
		output.WriteString("Latest version: (not yet checked)\n")
	}

	output.WriteString(fmt.Sprintf("Repository: %s\n", status.Repo))
	output.WriteString(fmt.Sprintf("Channel: %s\n", status.Channel))
	output.WriteString(fmt.Sprintf("Updates enabled: %v\n", status.Enabled))
	output.WriteString(fmt.Sprintf("\nRelease notes: %s\n", status.ReleaseURL))

	if status.LatestVersion != "" {
		output.WriteString("\n💡 To update, ask: \"Please update to the latest version\"\n")
	}

	return jsonrpc.NewResponse(id, textContent(output.String()))
}
