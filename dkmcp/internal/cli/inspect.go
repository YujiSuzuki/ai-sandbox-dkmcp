// inspect.go implements the 'inspect' command for displaying container details.
// It shows configuration, network settings, and mount information for a container.
//
// inspect.goはコンテナの詳細を表示する'inspect'コマンドを実装します。
// 設定、ネットワーク設定、マウント情報を表示します。
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

// inspectJSONFlag controls whether to output JSON instead of summary.
// inspectJSONFlagはサマリーの代わりにJSONを出力するかどうかを制御します。
var inspectJSONFlag bool

// inspectCmd represents the 'inspect' command for displaying container details.
// If no container is specified and current container is set, uses the current container.
//
// inspectCmdはコンテナ詳細を表示する'inspect'コマンドを表します。
// コンテナが指定されていない場合、カレントコンテナが設定されていればそれを使用します。
var inspectCmd = &cobra.Command{
	Use:   "inspect [container]",
	Short: "Get detailed information about a container",
	Long: `Display detailed information about a container including configuration,
network settings, and mount information.

By default, shows a human-readable summary. Use --json for full JSON output.

If no container is specified and a current container is set (via 'dkmcp use'),
it will use the current container.

Examples:
  dkmcp inspect securenote-api         # Show summary (default)
  dkmcp inspect securenote-api --json  # Show full JSON
  dkmcp use securenote-api             # Set current container
  dkmcp inspect                        # Inspect current container`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInspect,
}

// init registers the inspect command with the root command.
// This function is automatically called when the package is imported.
//
// initはinspectコマンドをルートコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	rootCmd.AddCommand(inspectCmd)

	// Add --json flag for full JSON output.
	// フルJSON出力用の--jsonフラグを追加します。
	inspectCmd.Flags().BoolVar(&inspectJSONFlag, "json", false, "Output full JSON instead of summary")
}

// runInspect is the execution function for the inspect command.
// It retrieves and displays detailed information for the specified container.
//
// runInspectはinspectコマンドの実行関数です。
// 指定されたコンテナの詳細情報を取得して表示します。
func runInspect(cmd *cobra.Command, args []string) error {
	// Determine container name (from argument or current)
	// コンテナ名を決定（引数またはカレントから）
	var containerArg string
	if len(args) > 0 {
		containerArg = args[0]
	}

	container, err := GetContainerOrCurrent(containerArg)
	if err != nil {
		return err
	}

	// Create a DirectBackend for Docker access.
	// Docker接続用のDirectBackendを作成します。
	backend, err := NewDirectBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	// Retrieve detailed information from the container.
	// コンテナから詳細情報を取得します。
	ctx := context.Background()
	info, err := backend.docker.InspectContainer(ctx, container)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Output based on --json flag
	// --jsonフラグに基づいて出力
	if inspectJSONFlag {
		// JSON output mode
		// JSON出力モード
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format container info: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Summary output mode (default)
		// サマリー出力モード（デフォルト）
		printInspectSummary(container, info)
	}

	return nil
}

// printInspectSummary prints a human-readable summary of the container.
// It extracts key information like state, IP addresses, and ports.
//
// printInspectSummaryはコンテナの人が読めるサマリーを出力します。
// 状態、IPアドレス、ポートなどの重要な情報を抽出します。
func printInspectSummary(containerName string, info *types.ContainerJSON) {
	fmt.Printf("=== Container Summary: %s ===\n\n", containerName)

	// State information
	// 状態情報
	if info.State != nil {
		fmt.Printf("State:    %s\n", info.State.Status)
		if info.State.Running {
			fmt.Printf("Started:  %s\n", info.State.StartedAt)
		}
	}

	// Image information
	// イメージ情報
	if info.Config != nil {
		fmt.Printf("Image:    %s\n", info.Config.Image)
	}

	// Network information - IP addresses
	// ネットワーク情報 - IPアドレス
	fmt.Println("\n--- Network ---")
	printNetworkSummary(info)

	// Port mappings
	// ポートマッピング
	if info.NetworkSettings != nil && len(info.NetworkSettings.Ports) > 0 {
		fmt.Println("\n--- Ports ---")
		printPortSummary(info.NetworkSettings.Ports)
	}

	// Mount information
	// マウント情報
	if len(info.Mounts) > 0 {
		fmt.Println("\n--- Mounts ---")
		printMountSummary(info.Mounts)
	}
}

// printNetworkSummary prints network information including IP addresses.
// printNetworkSummaryはIPアドレスを含むネットワーク情報を出力します。
func printNetworkSummary(info *types.ContainerJSON) {
	if info.NetworkSettings == nil || len(info.NetworkSettings.Networks) == 0 {
		fmt.Println("  (no networks)")
		return
	}

	// Sort network names for consistent output
	// 一貫した出力のためにネットワーク名をソート
	var networkNames []string
	for name := range info.NetworkSettings.Networks {
		networkNames = append(networkNames, name)
	}
	sort.Strings(networkNames)

	for _, name := range networkNames {
		network := info.NetworkSettings.Networks[name]
		fmt.Printf("  %s:\n", name)
		if network.IPAddress != "" {
			fmt.Printf("    IP:      %s\n", network.IPAddress)
		}
		if network.Gateway != "" {
			fmt.Printf("    Gateway: %s\n", network.Gateway)
		}
		if network.MacAddress != "" {
			fmt.Printf("    MAC:     %s\n", network.MacAddress)
		}
	}
}

// printPortSummary prints port binding information.
// printPortSummaryはポートバインディング情報を出力します。
func printPortSummary(ports nat.PortMap) {
	if len(ports) == 0 {
		fmt.Println("  (no ports)")
		return
	}

	// Sort port keys for consistent output
	// 一貫した出力のためにポートキーをソート
	var portKeys []string
	for port := range ports {
		portKeys = append(portKeys, string(port))
	}
	sort.Strings(portKeys)

	for _, key := range portKeys {
		port := nat.Port(key)
		bindings := ports[port]
		if len(bindings) == 0 {
			fmt.Printf("  %s (exposed, not bound)\n", key)
		} else {
			for _, binding := range bindings {
				hostIP := binding.HostIP
				if hostIP == "" {
					hostIP = "0.0.0.0"
				}
				fmt.Printf("  %s:%s -> %s\n", hostIP, binding.HostPort, key)
			}
		}
	}
}

// printMountSummary prints a brief summary of container mounts.
// printMountSummaryはコンテナマウントの簡潔なサマリーを出力します。
func printMountSummary(mounts []types.MountPoint) {
	for _, mount := range mounts {
		var mountInfo strings.Builder
		mountInfo.WriteString(fmt.Sprintf("  %s -> %s", mount.Source, mount.Destination))
		if mount.RW {
			mountInfo.WriteString(" (rw)")
		} else {
			mountInfo.WriteString(" (ro)")
		}
		fmt.Println(mountInfo.String())
	}
}
