// list.go implements the 'list' command for displaying accessible Docker containers.
// It uses the DirectBackend to communicate with Docker and applies the security policy.
//
// list.goはアクセス可能なDockerコンテナを表示する'list'コマンドを実装します。
// DirectBackendを使用してDockerと通信し、セキュリティポリシーを適用します。
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/docker"
)

// listCmd represents the 'list' command for listing accessible containers.
// It shows containers that match the allowed_containers patterns in the security policy.
//
// listCmdはアクセス可能なコンテナを一覧表示する'list'コマンドを表します。
// セキュリティポリシーのallowed_containersパターンに一致するコンテナを表示します。
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List accessible containers",
	Long:  `List all Docker containers that are accessible according to the security policy.`,
	RunE:  runList,
}

// init registers the list command with the root command.
// This function is automatically called when the package is imported.
//
// initはlistコマンドをルートコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	rootCmd.AddCommand(listCmd)
}

// runList is the execution function for the list command.
// It creates a DirectBackend, retrieves the container list, and displays it in a table.
//
// runListはlistコマンドの実行関数です。
// DirectBackendを作成し、コンテナリストを取得して、テーブル形式で表示します。
func runList(cmd *cobra.Command, args []string) error {
	// Create a DirectBackend for Docker access.
	// Docker接続用のDirectBackendを作成します。
	backend, err := NewDirectBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	// Retrieve the list of accessible containers.
	// アクセス可能なコンテナのリストを取得します。
	ctx := context.Background()
	containers, err := backend.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Display the containers in a formatted table.
	// コンテナをフォーマットされたテーブルで表示します。
	return printContainerTable(containers)
}

// printContainerTable prints containers in a formatted table.
// It uses tabwriter for aligned column output.
// If no containers are found, it prints a message indicating this.
//
// printContainerTableはコンテナをフォーマットされたテーブルで表示します。
// 列を揃えた出力のためにtabwriterを使用します。
// コンテナが見つからない場合は、その旨のメッセージを表示します。
func printContainerTable(containers []docker.ContainerInfo) error {
	// Handle the case of no accessible containers.
	// アクセス可能なコンテナがない場合を処理します。
	if len(containers) == 0 {
		fmt.Println("No accessible containers found.")
		return nil
	}

	// Create a tabwriter for formatted table output.
	// Parameters: output, minwidth, tabwidth, padding, padchar, flags
	//
	// フォーマットされたテーブル出力用のtabwriterを作成します。
	// パラメータ：出力、最小幅、タブ幅、パディング、パディング文字、フラグ
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print table header.
	// テーブルヘッダーを出力します。
	fmt.Fprintln(w, "NAME\tID\tIMAGE\tSTATE\tSTATUS\tPORTS")
	fmt.Fprintln(w, "----\t--\t-----\t-----\t------\t-----")

	// Print each container's information.
	// 各コンテナの情報を出力します。
	for _, c := range containers {
		// Format ports as comma-separated string, truncate if too long.
		// ポートをカンマ区切りの文字列にフォーマットし、長すぎる場合は切り詰めます。
		ports := formatPortsForDisplay(c.Ports)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Name, c.ID, c.Image, c.State, c.Status, ports)
	}

	// Flush the tabwriter to ensure all output is written.
	// すべての出力が書き込まれるようにtabwriterをフラッシュします。
	w.Flush()
	return nil
}

// formatPortsForDisplay formats port list for CLI table display.
// If no ports, returns "-". If too long, truncates with "...".
//
// formatPortsForDisplayはCLIテーブル表示用にポートリストをフォーマットします。
// ポートがない場合は"-"を返します。長すぎる場合は"..."で切り詰めます。
func formatPortsForDisplay(ports []string) string {
	if len(ports) == 0 {
		return "-"
	}

	// Join ports with comma separator.
	// ポートをカンマで結合します。
	portsStr := strings.Join(ports, ", ")

	// Truncate if too long (max 40 characters for readability).
	// 読みやすさのため、長すぎる場合は切り詰めます（最大40文字）。
	const maxLen = 40
	if len(portsStr) > maxLen {
		portsStr = portsStr[:maxLen-3] + "..."
	}

	return portsStr
}
