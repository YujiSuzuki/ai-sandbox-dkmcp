// exec.go implements the 'exec' command for executing commands in containers.
// It enforces security by only allowing whitelisted commands defined in the security policy.
//
// exec.goはコンテナ内でコマンドを実行する'exec'コマンドを実装します。
// セキュリティポリシーで定義されたホワイトリストに登録されたコマンドのみを許可してセキュリティを強制します。
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// execContainer specifies the target container via -c flag.
	// When not set, the current container is used.
	//
	// execContainerは-cフラグでターゲットコンテナを指定します。
	// 設定されていない場合、カレントコンテナが使用されます。
	execContainer string
)

// execCmd represents the 'exec' command for executing commands in containers.
// If no container is specified via -c flag and current container is set, uses the current container.
//
// execCmdはコンテナ内でコマンドを実行する'exec'コマンドを表します。
// -cフラグでコンテナが指定されていない場合、カレントコンテナが設定されていればそれを使用します。
var execCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Execute a command in a container",
	Long: `Execute a command inside a container. Only whitelisted commands are allowed
based on the security policy configuration.

If no container is specified via -c flag and a current container is set (via 'dkmcp use'),
it will use the current container.

Examples:
  dkmcp exec -c securenote-api "npm test"  # Execute in specific container
  dkmcp use securenote-api                 # Set current container
  dkmcp exec "npm test"                    # Execute in current container
  dkmcp exec --dangerously "tail -f /var/log/app.log"  # Dangerous mode`,
	Args: cobra.MinimumNArgs(1),
	RunE: runExec,
}

// init registers the exec command with the root command.
// This function is automatically called when the package is imported.
//
// initはexecコマンドをルートコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	rootCmd.AddCommand(execCmd)

	// Add -c flag for specifying container
	// コンテナ指定用の-cフラグを追加
	execCmd.Flags().StringVarP(&execContainer, "container", "c", "", "Target container (uses current container if not specified)")

	// Add --dangerously flag for dangerous mode execution
	// 危険モード実行用の--dangerouslyフラグを追加
	execCmd.Flags().Bool("dangerously", false, "Enable dangerous mode to execute commands from exec_dangerously list (file paths are still checked against blocked_paths)")
}

// runExec is the execution function for the exec command.
// It executes the specified command in the container and prints the result.
// Returns an error if the command is not whitelisted or execution fails.
//
// runExecはexecコマンドの実行関数です。
// 指定されたコマンドをコンテナ内で実行し、結果を出力します。
// コマンドがホワイトリストに登録されていないか、実行に失敗した場合はエラーを返します。
func runExec(cmd *cobra.Command, args []string) error {
	// Determine container name (from -c flag or current)
	// コンテナ名を決定（-cフラグまたはカレントから）
	container, err := GetContainerOrCurrent(execContainer)
	if err != nil {
		return err
	}

	// Join all arguments to form the complete command.
	// This allows commands with spaces like "npm test" or "ls -la".
	//
	// すべての引数を結合して完全なコマンドを形成します。
	// これにより"npm test"や"ls -la"のようなスペースを含むコマンドが可能になります。
	command := strings.Join(args, " ")

	// Get the dangerously flag value.
	// dangerouslyフラグの値を取得します。
	dangerously, _ := cmd.Flags().GetBool("dangerously")

	// Create a DirectBackend for Docker access.
	// Docker接続用のDirectBackendを作成します。
	backend, err := NewDirectBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	// Execute the command in the container.
	// If dangerously is true, exec_dangerously commands are allowed.
	//
	// コンテナ内でコマンドを実行します。
	// dangerouslyがtrueの場合、exec_dangerouslyコマンドが許可されます。
	ctx := context.Background()
	result, err := backend.Exec(ctx, container, command, dangerously)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Print the exit code and output.
	// 終了コードと出力を表示します。
	fmt.Printf("Exit Code: %d\n\n", result.ExitCode)
	fmt.Println(result.Output)

	// Return error if command exited with non-zero status.
	// This allows scripts to detect command failures.
	//
	// コマンドが非ゼロステータスで終了した場合はエラーを返します。
	// これによりスクリプトがコマンドの失敗を検出できます。
	if result.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", result.ExitCode)
	}

	return nil
}
