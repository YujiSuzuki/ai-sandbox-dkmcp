// stats.go implements the 'stats' command for displaying container resource statistics.
// It shows CPU, memory, network, and disk I/O usage for a container.
//
// stats.goはコンテナのリソース統計を表示する'stats'コマンドを実装します。
// CPU、メモリ、ネットワーク、ディスクI/Oの使用状況を表示します。
package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// statsCmd represents the 'stats' command for displaying container statistics.
// If no container is specified and current container is set, uses the current container.
//
// statsCmdはコンテナ統計を表示する'stats'コマンドを表します。
// コンテナが指定されていない場合、カレントコンテナが設定されていればそれを使用します。
var statsCmd = &cobra.Command{
	Use:   "stats [container]",
	Short: "Get resource statistics from a container",
	Long: `Display resource statistics (CPU, memory, network, disk I/O) for a container.

If no container is specified and a current container is set (via 'dkmcp use'),
it will use the current container.

Examples:
  dkmcp stats securenote-api    # Stats for specific container
  dkmcp use securenote-api      # Set current container
  dkmcp stats                   # Stats for current container`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStats,
}

// init registers the stats command with the root command.
// This function is automatically called when the package is imported.
//
// initはstatsコマンドをルートコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	rootCmd.AddCommand(statsCmd)
}

// runStats is the execution function for the stats command.
// It retrieves and displays resource statistics for the specified container.
//
// runStatsはstatsコマンドの実行関数です。
// 指定されたコンテナのリソース統計を取得して表示します。
func runStats(cmd *cobra.Command, args []string) error {
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

	// Retrieve statistics from the container.
	// コンテナから統計を取得します。
	ctx := context.Background()
	stats, err := backend.docker.GetStats(ctx, container)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	// Format stats as pretty-printed JSON
	// 統計情報を整形されたJSONでフォーマット
	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format stats: %w", err)
	}

	// Print the statistics.
	// 統計を出力します。
	fmt.Printf("Resource statistics for container '%s':\n\n", container)
	fmt.Println(string(jsonData))

	return nil
}
