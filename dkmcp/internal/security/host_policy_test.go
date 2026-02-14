package security

import (
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

func newTestHostConfig() *config.HostCommandsConfig {
	return &config.HostCommandsConfig{
		Enabled:           true,
		AllowedContainers: []string{"securenote-*", "demo-*"},
		AllowedProjects:   []string{"demo-apps"},
		Whitelist: map[string][]string{
			"docker": {"ps", "logs *", "stats *", "inspect *"},
			"git":    {"status", "diff *", "log --oneline *"},
			"df":     {"-h"},
			"free":   {"-m"},
		},
		Deny: map[string][]string{
			"docker": {"rm *", "rmi *", "system prune *"},
		},
		Dangerously: config.HostCommandsDangerously{
			Enabled: true,
			Commands: map[string][]string{
				"docker": {"restart", "stop", "start"},
				"git":    {"checkout", "pull"},
			},
		},
	}
}

// --- Normal mode tests ---

func TestHostCommandPolicy_NormalMode_Allowed(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"docker ps", "docker ps"},
		{"docker logs with container", "docker logs securenote-api"},
		{"git status", "git status"},
		{"git diff", "git diff HEAD"},
		{"df -h", "df -h"},
		{"free -m", "free -m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommand(tt.command)
			if err != nil {
				t.Errorf("CanExecHostCommand(%q) error = %v", tt.command, err)
			}
			if !ok {
				t.Errorf("CanExecHostCommand(%q) = false, want true", tt.command)
			}
		})
	}
}

func TestHostCommandPolicy_NormalMode_Denied(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"not in whitelist", "curl http://localhost"},
		{"docker rm denied", "docker rm securenote-api"},
		{"docker rmi denied", "docker rmi myimage"},
		{"docker system prune denied", "docker system prune -a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommand(tt.command)
			if ok {
				t.Errorf("CanExecHostCommand(%q) = true, want false", tt.command)
			}
			if err == nil {
				t.Errorf("CanExecHostCommand(%q) should return error", tt.command)
			}
		})
	}
}

func TestHostCommandPolicy_NormalMode_PipeRejected(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []string{
		"docker ps | grep api",
		"git status > /tmp/out",
		"docker logs api; rm -rf /",
		"docker ps && echo hacked",
		"git diff $(cat /etc/passwd)",
		"git diff `cat /etc/passwd`",
		"git diff HEAD\nrm -rf /",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			ok, err := p.CanExecHostCommand(cmd)
			if ok {
				t.Errorf("CanExecHostCommand(%q) should reject pipes/redirects", cmd)
			}
			if err == nil {
				t.Error("should return error")
			}
		})
	}
}

func TestHostCommandPolicy_NormalMode_HintForDangerous(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	_, err := p.CanExecHostCommand("docker restart securenote-api")
	if err == nil {
		t.Error("should return error")
	}
	// Error message should hint about dangerously mode
	if err != nil {
		errMsg := err.Error()
		if !searchSubstring(errMsg, "dangerously") {
			t.Errorf("error should mention dangerously hint, got: %s", errMsg)
		}
	}
}

// --- Dangerous mode tests ---

func TestHostCommandPolicy_DangerousMode_Allowed(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"docker restart", "docker restart securenote-api"},
		{"docker stop", "docker stop securenote-api"},
		{"git checkout", "git checkout main"},
		{"git pull", "git pull origin main"},
		// Normal whitelist commands should also work in dangerous mode
		{"docker ps via dangerous", "docker ps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommandDangerously(tt.command)
			if err != nil {
				t.Errorf("CanExecHostCommandDangerously(%q) error = %v", tt.command, err)
			}
			if !ok {
				t.Errorf("CanExecHostCommandDangerously(%q) = false, want true", tt.command)
			}
		})
	}
}

func TestHostCommandPolicy_DangerousMode_Denied(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"not in any list", "curl http://localhost"},
		{"docker rm still denied", "docker rm securenote-api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommandDangerously(tt.command)
			if ok {
				t.Errorf("CanExecHostCommandDangerously(%q) = true, want false", tt.command)
			}
			if err == nil {
				t.Errorf("CanExecHostCommandDangerously(%q) should return error", tt.command)
			}
		})
	}
}

func TestHostCommandPolicy_DangerousMode_Disabled(t *testing.T) {
	cfg := newTestHostConfig()
	cfg.Dangerously.Enabled = false
	p := NewHostCommandPolicy(cfg)

	_, err := p.CanExecHostCommandDangerously("docker restart securenote-api")
	if err == nil {
		t.Error("should return error when dangerous mode is disabled")
	}
}

func TestHostCommandPolicy_DangerousMode_PathTraversal(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	_, err := p.CanExecHostCommandDangerously("git checkout ../../etc/passwd")
	if err == nil {
		t.Error("should reject path traversal in dangerous mode")
	}
}

// --- Disabled tests ---

func TestHostCommandPolicy_Disabled(t *testing.T) {
	cfg := newTestHostConfig()
	cfg.Enabled = false
	p := NewHostCommandPolicy(cfg)

	_, err := p.CanExecHostCommand("docker ps")
	if err == nil {
		t.Error("should return error when host commands are disabled")
	}

	_, err = p.CanExecHostCommandDangerously("docker ps")
	if err == nil {
		t.Error("should return error when host commands are disabled")
	}
}

// --- GetAllowedHostCommands ---

func TestHostCommandPolicy_GetAllowedHostCommands(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())
	cmds := p.GetAllowedHostCommands()

	if len(cmds) != 4 {
		t.Errorf("GetAllowedHostCommands() returned %d commands, want 4", len(cmds))
	}

	if _, ok := cmds["docker"]; !ok {
		t.Error("should include docker commands")
	}
	if _, ok := cmds["git"]; !ok {
		t.Error("should include git commands")
	}
}

// --- Wildcard pattern tests ---

func TestHostCommandPolicy_WildcardPatterns(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	// "logs *" should match any container name
	ok, err := p.CanExecHostCommand("docker logs securenote-api")
	if err != nil || !ok {
		t.Errorf("should match 'logs *' pattern: ok=%v, err=%v", ok, err)
	}

	// "inspect *" should match any container name
	ok, err = p.CanExecHostCommand("docker inspect securenote-api")
	if err != nil || !ok {
		t.Errorf("should match 'inspect *' pattern: ok=%v, err=%v", ok, err)
	}
}

// --- Container restriction tests ---

func TestHostCommandPolicy_ContainerRestrictions(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	// Allowed container (matches "securenote-*")
	ok, err := p.CanExecHostCommand("docker logs securenote-api")
	if err != nil || !ok {
		t.Errorf("should allow securenote-api: ok=%v, err=%v", ok, err)
	}

	// Allowed container (matches "demo-*")
	ok, err = p.CanExecHostCommand("docker logs demo-web")
	if err != nil || !ok {
		t.Errorf("should allow demo-web: ok=%v, err=%v", ok, err)
	}

	// Disallowed container
	ok, err = p.CanExecHostCommand("docker logs malicious-container")
	if ok {
		t.Error("should reject container not in allowed list")
	}
}

// --- Helper functions ---

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
