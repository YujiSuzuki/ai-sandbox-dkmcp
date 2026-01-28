// inspect_test.go contains unit tests for the inspect command summary functions.
// These tests verify that container information is formatted correctly for display.
//
// inspect_test.goはinspectコマンドのサマリー関数のユニットテストを含みます。
// これらのテストはコンテナ情報が表示用に正しくフォーマットされることを確認します。
package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// captureOutput captures stdout during function execution.
// captureOutputは関数実行中のstdoutをキャプチャします。
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestPrintNetworkSummary tests the network summary output formatting.
// TestPrintNetworkSummaryはネットワークサマリー出力のフォーマットをテストします。
func TestPrintNetworkSummary(t *testing.T) {
	tests := []struct {
		name     string             // Test case name / テストケース名
		info     *types.ContainerJSON
		contains []string           // Strings that should be in output / 出力に含まれるべき文字列
	}{
		{
			name: "nil network settings",
			info: &types.ContainerJSON{
				NetworkSettings: nil,
			},
			contains: []string{"(no networks)"},
		},
		{
			name: "empty networks",
			info: &types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
				},
			},
			contains: []string{"(no networks)"},
		},
		{
			name: "single network with IP",
			info: &types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"bridge": {
							IPAddress: "172.17.0.2",
							Gateway:   "172.17.0.1",
						},
					},
				},
			},
			contains: []string{"bridge:", "IP:", "172.17.0.2", "Gateway:", "172.17.0.1"},
		},
		{
			name: "network with MAC address",
			info: &types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"custom": {
							IPAddress:  "10.0.0.5",
							MacAddress: "02:42:0a:00:00:05",
						},
					},
				},
			},
			contains: []string{"custom:", "IP:", "10.0.0.5", "MAC:", "02:42:0a:00:00:05"},
		},
		{
			name: "multiple networks sorted",
			info: &types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"zebra": {IPAddress: "10.0.2.1"},
						"alpha": {IPAddress: "10.0.1.1"},
					},
				},
			},
			contains: []string{"alpha:", "10.0.1.1", "zebra:", "10.0.2.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printNetworkSummary(tt.info)
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, output)
				}
			}
		})
	}
}

// TestPrintPortSummary tests the port summary output formatting.
// TestPrintPortSummaryはポートサマリー出力のフォーマットをテストします。
func TestPrintPortSummary(t *testing.T) {
	tests := []struct {
		name     string      // Test case name / テストケース名
		ports    nat.PortMap // Input ports / 入力ポート
		contains []string    // Strings that should be in output / 出力に含まれるべき文字列
	}{
		{
			name:     "empty ports",
			ports:    nat.PortMap{},
			contains: []string{"(no ports)"},
		},
		{
			name:     "nil ports",
			ports:    nil,
			contains: []string{"(no ports)"},
		},
		{
			name: "single bound port",
			ports: nat.PortMap{
				"80/tcp": []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: "8080"},
				},
			},
			contains: []string{"0.0.0.0:8080 -> 80/tcp"},
		},
		{
			name: "port with empty host IP defaults to 0.0.0.0",
			ports: nat.PortMap{
				"443/tcp": []nat.PortBinding{
					{HostIP: "", HostPort: "443"},
				},
			},
			contains: []string{"0.0.0.0:443 -> 443/tcp"},
		},
		{
			name: "exposed but not bound port",
			ports: nat.PortMap{
				"3306/tcp": []nat.PortBinding{},
			},
			contains: []string{"3306/tcp (exposed, not bound)"},
		},
		{
			name: "localhost binding",
			ports: nat.PortMap{
				"5432/tcp": []nat.PortBinding{
					{HostIP: "127.0.0.1", HostPort: "5432"},
				},
			},
			contains: []string{"127.0.0.1:5432 -> 5432/tcp"},
		},
		{
			name: "multiple bindings for same port",
			ports: nat.PortMap{
				"80/tcp": []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: "80"},
					{HostIP: "::", HostPort: "80"},
				},
			},
			contains: []string{"0.0.0.0:80 -> 80/tcp", ":::80 -> 80/tcp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printPortSummary(tt.ports)
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, output)
				}
			}
		})
	}
}

// TestPrintMountSummary tests the mount summary output formatting.
// TestPrintMountSummaryはマウントサマリー出力のフォーマットをテストします。
func TestPrintMountSummary(t *testing.T) {
	tests := []struct {
		name     string             // Test case name / テストケース名
		mounts   []types.MountPoint // Input mounts / 入力マウント
		contains []string           // Strings that should be in output / 出力に含まれるべき文字列
	}{
		{
			name:     "empty mounts",
			mounts:   []types.MountPoint{},
			contains: []string{}, // No output for empty mounts
		},
		{
			name: "single read-write mount",
			mounts: []types.MountPoint{
				{
					Source:      "/host/data",
					Destination: "/container/data",
					RW:          true,
				},
			},
			contains: []string{"/host/data -> /container/data", "(rw)"},
		},
		{
			name: "single read-only mount",
			mounts: []types.MountPoint{
				{
					Source:      "/host/config",
					Destination: "/container/config",
					RW:          false,
				},
			},
			contains: []string{"/host/config -> /container/config", "(ro)"},
		},
		{
			name: "multiple mounts",
			mounts: []types.MountPoint{
				{
					Source:      "/data",
					Destination: "/app/data",
					RW:          true,
				},
				{
					Source:      "/secrets",
					Destination: "/app/secrets",
					RW:          false,
				},
			},
			contains: []string{"/data -> /app/data", "(rw)", "/secrets -> /app/secrets", "(ro)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printMountSummary(tt.mounts)
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, output)
				}
			}
		})
	}
}

// TestPrintInspectSummary tests the full inspect summary output.
// TestPrintInspectSummaryは完全なinspectサマリー出力をテストします。
func TestPrintInspectSummary(t *testing.T) {
	tests := []struct {
		name          string             // Test case name / テストケース名
		containerName string             // Container name / コンテナ名
		info          *types.ContainerJSON
		contains      []string           // Strings that should be in output / 出力に含まれるべき文字列
	}{
		{
			name:          "running container",
			containerName: "test-container",
			info: &types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status:    "running",
						Running:   true,
						StartedAt: "2024-01-15T10:30:00Z",
					},
				},
				Config: &container.Config{
					Image: "nginx:latest",
				},
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"bridge": {IPAddress: "172.17.0.2"},
					},
				},
			},
			contains: []string{
				"=== Container Summary: test-container ===",
				"State:", "running",
				"Started:", "2024-01-15T10:30:00Z",
				"Image:", "nginx:latest",
				"--- Network ---",
				"bridge:", "172.17.0.2",
			},
		},
		{
			name:          "stopped container",
			containerName: "stopped-container",
			info: &types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status:  "exited",
						Running: false,
					},
				},
				Config: &container.Config{
					Image: "alpine:3.18",
				},
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
				},
			},
			contains: []string{
				"=== Container Summary: stopped-container ===",
				"State:", "exited",
				"Image:", "alpine:3.18",
			},
		},
		{
			name:          "container with ports",
			containerName: "web-server",
			info: &types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status:  "running",
						Running: true,
					},
				},
				Config: &container.Config{
					Image: "nginx:latest",
				},
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{
								{HostIP: "0.0.0.0", HostPort: "8080"},
							},
						},
					},
				},
			},
			contains: []string{
				"--- Ports ---",
				"0.0.0.0:8080 -> 80/tcp",
			},
		},
		{
			name:          "container with mounts",
			containerName: "data-container",
			info: &types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status:  "running",
						Running: true,
					},
				},
				Config: &container.Config{
					Image: "postgres:15",
				},
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
				},
				Mounts: []types.MountPoint{
					{
						Source:      "/var/lib/postgres",
						Destination: "/var/lib/postgresql/data",
						RW:          true,
					},
				},
			},
			contains: []string{
				"--- Mounts ---",
				"/var/lib/postgres -> /var/lib/postgresql/data",
				"(rw)",
			},
		},
		{
			name:          "container with nil state",
			containerName: "nil-state",
			info: &types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: nil,
				},
				Config: &container.Config{
					Image: "busybox",
				},
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
				},
			},
			contains: []string{
				"=== Container Summary: nil-state ===",
				"Image:", "busybox",
			},
		},
		{
			name:          "container with nil config",
			containerName: "nil-config",
			info: &types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "running",
					},
				},
				Config: nil,
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
				},
			},
			contains: []string{
				"=== Container Summary: nil-config ===",
				"State:", "running",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printInspectSummary(tt.containerName, tt.info)
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, output)
				}
			}
		})
	}
}
