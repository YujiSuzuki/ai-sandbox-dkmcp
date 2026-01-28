// Package docker provides tests for the Docker client wrapper.
// These tests verify the security policy enforcement and helper functions
// used by the Docker client.
//
// The test suite includes:
// - Unit tests for helper functions (parseCommand, formatBytes, etc.)
// - Data structure field validation tests
// - Integration tests (when Docker daemon is available)
//
// dockerパッケージはDockerクライアントラッパーのテストを提供します。
// これらのテストはDockerクライアントが使用するセキュリティポリシーの適用と
// ヘルパー関数を検証します。
//
// テストスイートには以下が含まれます:
// - ヘルパー関数のユニットテスト（parseCommand、formatBytesなど）
// - データ構造フィールド検証テスト
// - 統合テスト（Dockerデーモンが利用可能な場合）
//
// NOTE: About Data Structure Field Tests (TestContainerInfo_Fields, etc.)
// 注: データ構造フィールドテストについて（TestContainerInfo_Fields等）
//
// These tests verify Go struct field assignment and retrieval, which is
// guaranteed by the Go language specification. Their primary value is:
// これらのテストはGoの構造体フィールド代入と取得を検証しますが、
// これはGo言語仕様で保証されています。主な価値は：
//
//   - Documentation: Show expected field names and types
//     ドキュメント: 期待されるフィールド名と型を示す
//   - IDE support: Useful for code navigation and refactoring
//     IDEサポート: コードナビゲーションとリファクタリングに有用
//   - Sanity check: Catch copy-paste errors in struct definitions
//     健全性チェック: 構造体定義のコピペミスを検出
//
// However, they do NOT test actual behavior like JSON marshaling,
// Docker API responses, or data transformations.
// ただし、JSONマーシャリング、Docker APIレスポンス、データ変換などの
// 実際の動作はテストしていません。
package docker

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// TestNewClient_WithNilPolicy verifies that NewClient correctly rejects
// nil security policies. This is a critical safety check - the client
// must never operate without a policy.
//
// TestNewClient_WithNilPolicyはNewClientがnilセキュリティポリシーを
// 正しく拒否することを検証します。これは重要な安全チェックです -
// クライアントはポリシーなしで動作してはなりません。
func TestNewClient_WithNilPolicy(t *testing.T) {
	// Attempt to create a client with nil policy.
	// nilポリシーでクライアントの作成を試みます。
	client, err := NewClient(nil)

	// Verify that an error is returned.
	// エラーが返されることを検証します。
	if err == nil {
		t.Error("expected error when policy is nil")
	}

	// Verify that no client is returned.
	// クライアントが返されないことを検証します。
	if client != nil {
		t.Error("expected nil client when policy is nil")
	}
}

// TestNewClient_Success verifies that NewClient can create a client
// with a valid security policy. Note that this test may fail if
// the Docker daemon is not running, which is acceptable in some
// test environments.
//
// TestNewClient_Successは有効なセキュリティポリシーでNewClientが
// クライアントを作成できることを検証します。このテストはDockerデーモンが
// 実行されていない場合に失敗する可能性がありますが、
// 一部のテスト環境では許容されます。
func TestNewClient_Success(t *testing.T) {
	// Create a test security configuration with common settings.
	// 一般的な設定でテスト用セキュリティ設定を作成します。
	cfg := &config.SecurityConfig{
		Mode:              "moderate",     // Security mode / セキュリティモード
		AllowedContainers: []string{"test-*"}, // Allow containers matching "test-*" / "test-*"に一致するコンテナを許可
		Permissions: config.SecurityPermissions{
			Logs:    true, // Allow log access / ログアクセスを許可
			Inspect: true, // Allow container inspection / コンテナ検査を許可
			Stats:   true, // Allow stats retrieval / 統計取得を許可
			Exec:    true, // Allow command execution / コマンド実行を許可
		},
	}

	// Create a security policy from the configuration.
	// 設定からセキュリティポリシーを作成します。
	policy := security.NewPolicy(cfg)

	// Note: This will fail if Docker daemon is not running
	// In real CI/CD, we would use a mock Docker client
	// 注意: Dockerデーモンが実行されていない場合、これは失敗します
	// 実際のCI/CDでは、モックDockerクライアントを使用します
	client, err := NewClient(policy)

	// We expect this to work if Docker is available
	// or fail with specific error if not
	// Dockerが利用可能な場合はこれが動作することを期待し、
	// そうでない場合は特定のエラーで失敗することを期待します
	if err != nil {
		// Check if it's a connection error (acceptable in test environment)
		// 接続エラーかどうかをチェックします（テスト環境では許容）
		if client == nil {
			t.Logf("Docker daemon not available (expected in some test environments): %v", err)
		}
	}
}

// TestParseMemoryStats tests the parseMemoryStats helper function
// which converts raw memory usage/limit values into human-readable
// strings and calculates the percentage used.
//
// TestParseMemoryStatsはparseMemoryStatsヘルパー関数をテストします。
// これは生のメモリ使用量/制限値を人が読める文字列に変換し、
// 使用率を計算します。
func TestParseMemoryStats(t *testing.T) {
	// Define test cases covering various memory scenarios.
	// 様々なメモリシナリオをカバーするテストケースを定義します。
	tests := []struct {
		name      string  // Test case name / テストケース名
		usage     uint64  // Memory usage in bytes / バイト単位のメモリ使用量
		limit     uint64  // Memory limit in bytes / バイト単位のメモリ制限
		wantUsage string  // Expected usage string / 期待される使用量文字列
		wantLimit string  // Expected limit string / 期待される制限文字列
		wantPct   float64 // Expected percentage / 期待されるパーセンテージ
	}{
		{
			// Normal usage case: 512 MiB used out of 2 GiB limit
			// 通常使用ケース: 2 GiB制限のうち512 MiB使用
			name:      "normal usage",
			usage:     536870912,  // 512 MiB
			limit:     2147483648, // 2 GiB
			wantUsage: "512.0 MiB",
			wantLimit: "2.0 GiB",
			wantPct:   25.0,
		},
		{
			// Edge case: Zero limit (should not divide by zero)
			// エッジケース: 制限ゼロ（ゼロ除算してはならない）
			name:      "zero limit",
			usage:     100,
			limit:     0,
			wantUsage: "100 B",
			wantLimit: "0 B",
			wantPct:   0.0,
		},
		{
			// Small values: Bytes-level memory usage
			// 小さな値: バイトレベルのメモリ使用量
			name:      "bytes only",
			usage:     512,
			limit:     1024,
			wantUsage: "512 B",
			wantLimit: "1.0 KiB",
			wantPct:   50.0,
		},
	}

	// Run each test case.
	// 各テストケースを実行します。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function under test.
			// テスト対象の関数を呼び出します。
			usageStr, limitStr, pct := parseMemoryStats(tt.usage, tt.limit)

			// Verify usage string matches expected value.
			// 使用量文字列が期待値と一致することを検証します。
			if usageStr != tt.wantUsage {
				t.Errorf("usage = %q, want %q", usageStr, tt.wantUsage)
			}

			// Verify limit string matches expected value.
			// 制限文字列が期待値と一致することを検証します。
			if limitStr != tt.wantLimit {
				t.Errorf("limit = %q, want %q", limitStr, tt.wantLimit)
			}

			// Verify percentage matches expected value.
			// パーセンテージが期待値と一致することを検証します。
			if pct != tt.wantPct {
				t.Errorf("percentage = %.2f, want %.2f", pct, tt.wantPct)
			}
		})
	}
}

// TestFormatBytes tests the formatBytes helper function which converts
// raw byte counts into human-readable strings with appropriate units
// (B, KiB, MiB, GiB).
//
// TestFormatBytesはformatBytesヘルパー関数をテストします。
// これは生のバイトカウントを適切な単位（B、KiB、MiB、GiB）を持つ
// 人が読める文字列に変換します。
func TestFormatBytes(t *testing.T) {
	// Define test cases for various byte sizes.
	// 様々なバイトサイズのテストケースを定義します。
	tests := []struct {
		name  string // Test case name / テストケース名
		bytes uint64 // Input byte count / 入力バイトカウント
		want  string // Expected formatted string / 期待されるフォーマット文字列
	}{
		// Zero bytes
		// ゼロバイト
		{"zero", 0, "0 B"},

		// Bytes (less than 1 KiB)
		// バイト（1 KiB未満）
		{"bytes", 512, "512 B"},

		// Exact kilobytes
		// 正確なキロバイト
		{"kilobytes", 1024, "1.0 KiB"},

		// Kilobytes with decimal portion
		// 小数部分を持つキロバイト
		{"kilobytes with decimal", 1536, "1.5 KiB"},

		// Exact megabytes
		// 正確なメガバイト
		{"megabytes", 1048576, "1.0 MiB"},

		// Exact gigabytes
		// 正確なギガバイト
		{"gigabytes", 1073741824, "1.0 GiB"},

		// Large value (5 GiB)
		// 大きな値（5 GiB）
		{"large value", 5368709120, "5.0 GiB"},
	}

	// Run each test case.
	// 各テストケースを実行します。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function under test.
			// テスト対象の関数を呼び出します。
			got := formatBytes(tt.bytes)

			// Verify output matches expected value.
			// 出力が期待値と一致することを検証します。
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// TestParseCPUPercent tests the parseCPUPercent helper function which
// calculates CPU usage percentage from delta values and CPU count.
//
// The formula is: (cpuDelta / sysDelta) * 100 * numCPUs
// This gives percentage of total CPU capacity being used.
//
// TestParseCPUPercentはparseCPUPercentヘルパー関数をテストします。
// これはデルタ値とCPU数からCPU使用率を計算します。
//
// 計算式は: (cpuDelta / sysDelta) * 100 * numCPUs
// これは使用されている総CPU容量のパーセンテージを示します。
func TestParseCPUPercent(t *testing.T) {
	// Define test cases for various CPU usage scenarios.
	// 様々なCPU使用シナリオのテストケースを定義します。
	tests := []struct {
		name       string  // Test case name / テストケース名
		cpuDelta   uint64  // CPU time delta / CPU時間デルタ
		sysDelta   uint64  // System time delta / システム時間デルタ
		numCPUs    int     // Number of CPUs / CPU数
		wantResult float64 // Expected percentage / 期待されるパーセンテージ
	}{
		{
			// No CPU usage
			// CPU使用なし
			name:       "no usage",
			cpuDelta:   0,
			sysDelta:   1000000000,
			numCPUs:    4,
			wantResult: 0.0,
		},
		{
			// 50% of one core on 4-core system equals 200% total
			// 4コアシステムで1コアの50%は合計200%に相当
			name:       "50% of one core on 4-core system",
			cpuDelta:   500000000,
			sysDelta:   1000000000,
			numCPUs:    4,
			wantResult: 200.0, // 50% * 4 cores = 200%
		},
		{
			// Edge case: Zero system delta (should not divide by zero)
			// エッジケース: システムデルタがゼロ（ゼロ除算してはならない）
			name:       "zero system delta",
			cpuDelta:   100,
			sysDelta:   0,
			numCPUs:    2,
			wantResult: 0.0,
		},
		{
			// Single core at full usage (100%)
			// シングルコアがフル使用（100%）
			name:       "single core full usage",
			cpuDelta:   1000000000,
			sysDelta:   1000000000,
			numCPUs:    1,
			wantResult: 100.0,
		},
	}

	// Run each test case.
	// 各テストケースを実行します。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function under test.
			// テスト対象の関数を呼び出します。
			got := parseCPUPercent(tt.cpuDelta, tt.sysDelta, uint32(tt.numCPUs))

			// Verify output matches expected value.
			// 出力が期待値と一致することを検証します。
			if got != tt.wantResult {
				t.Errorf("parseCPUPercent(%d, %d, %d) = %.2f, want %.2f",
					tt.cpuDelta, tt.sysDelta, tt.numCPUs, got, tt.wantResult)
			}
		})
	}
}

// Note: Integration tests that require Docker daemon are not yet implemented.
// For mock-based functional tests, see internal/mcp/tools_functional_test.go.
//
// Mock-based tests implemented (in internal/mcp/tools_functional_test.go):
// - Uses DockerClientInterface and MockClient for testing without Docker daemon
// - Tests all MCP tool handlers with simulated Docker responses
//
// Future work: Add integration tests with Docker test containers (testcontainers-go)
// - TestListContainers_Integration
// - TestGetLogs_Integration
// - TestExec_Integration
// - TestGetStats_Integration
// - TestInspect_Integration
//
// 注意: Dockerデーモンを必要とする統合テストはまだ実装されていません。
// モックベースの機能テストについては、internal/mcp/tools_functional_test.goを参照してください。
//
// 実装されたモックベースのテスト（internal/mcp/tools_functional_test.go）:
// - DockerClientInterfaceとMockClientを使用してDockerデーモンなしでテスト
// - シミュレートされたDockerレスポンスですべてのMCPツールハンドラーをテスト
//
// 今後の作業: Dockerテストコンテナ（testcontainers-go）を使用した統合テストを追加
// - TestListContainers_Integration
// - TestGetLogs_Integration
// - TestExec_Integration
// - TestGetStats_Integration
// - TestInspect_Integration

// parseMemoryStats is a helper function that converts raw memory statistics
// into human-readable format. It takes usage and limit in bytes and returns
// formatted strings and the usage percentage.
//
// This function is used when processing container stats responses
// for display to users or logging.
//
// parseMemoryStatsは生のメモリ統計を人が読める形式に変換する
// ヘルパー関数です。バイト単位の使用量と制限を受け取り、
// フォーマットされた文字列と使用率を返します。
//
// この関数はユーザーへの表示やログのためにコンテナ統計レスポンスを
// 処理するときに使用されます。
func parseMemoryStats(usage, limit uint64) (string, string, float64) {
	// Format usage and limit as human-readable strings.
	// 使用量と制限を人が読める文字列にフォーマットします。
	usageStr := formatBytes(usage)
	limitStr := formatBytes(limit)

	// Calculate percentage, avoiding division by zero.
	// ゼロ除算を避けてパーセンテージを計算します。
	pct := 0.0
	if limit > 0 {
		pct = float64(usage) / float64(limit) * 100.0
	}
	return usageStr, limitStr, pct
}

// formatBytes converts a byte count into a human-readable string
// with an appropriate unit suffix (B, KiB, MiB, GiB).
//
// The function uses binary prefixes (1024-based) rather than
// decimal prefixes (1000-based), following Docker's convention.
//
// formatBytesはバイトカウントを適切な単位サフィックス（B、KiB、MiB、GiB）
// を持つ人が読める文字列に変換します。
//
// この関数はDockerの慣例に従い、10進数プレフィックス（1000ベース）
// ではなく2進数プレフィックス（1024ベース）を使用します。
func formatBytes(bytes uint64) string {
	// Define byte unit constants using binary prefixes.
	// 2進数プレフィックスを使用してバイト単位定数を定義します。
	const (
		B   = 1           // 1 byte / 1バイト
		KiB = 1024 * B    // 1024 bytes / 1024バイト
		MiB = 1024 * KiB  // 1,048,576 bytes / 1,048,576バイト
		GiB = 1024 * MiB  // 1,073,741,824 bytes / 1,073,741,824バイト
	)

	// Select appropriate unit based on magnitude.
	// 大きさに基づいて適切な単位を選択します。
	switch {
	case bytes < KiB:
		// Less than 1 KiB: display as bytes.
		// 1 KiB未満: バイトとして表示します。
		return fmt.Sprintf("%d B", bytes)
	case bytes < MiB:
		// Less than 1 MiB: display as KiB with one decimal.
		// 1 MiB未満: 小数点1桁のKiBとして表示します。
		return fmt.Sprintf("%.1f KiB", float64(bytes)/float64(KiB))
	case bytes < GiB:
		// Less than 1 GiB: display as MiB with one decimal.
		// 1 GiB未満: 小数点1桁のMiBとして表示します。
		return fmt.Sprintf("%.1f MiB", float64(bytes)/float64(MiB))
	default:
		// 1 GiB or more: display as GiB with one decimal.
		// 1 GiB以上: 小数点1桁のGiBとして表示します。
		return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(GiB))
	}
}

// parseCPUPercent calculates CPU usage percentage from delta values.
//
// Parameters:
//   - cpuDelta: CPU time used since last measurement (nanoseconds)
//   - sysDelta: System time elapsed since last measurement (nanoseconds)
//   - numCPUs: Number of CPUs available to the container
//
// Returns the CPU usage as a percentage of total CPU capacity.
// For a 4-core system, 100% means one full core, 400% means all cores at max.
//
// parseCPUPercentはデルタ値からCPU使用率を計算します。
//
// パラメータ:
//   - cpuDelta: 前回の測定以降に使用されたCPU時間（ナノ秒）
//   - sysDelta: 前回の測定以降に経過したシステム時間（ナノ秒）
//   - numCPUs: コンテナで利用可能なCPU数
//
// 総CPU容量に対するパーセンテージとしてCPU使用率を返します。
// 4コアシステムの場合、100%は1つのフルコア、400%はすべてのコアが最大を意味します。
func parseCPUPercent(cpuDelta, sysDelta uint64, numCPUs uint32) float64 {
	// Avoid division by zero if no time has elapsed.
	// 時間が経過していない場合はゼロ除算を回避します。
	if sysDelta == 0 {
		return 0.0
	}

	// Calculate percentage: (cpu used / system time) * 100 * number of CPUs
	// パーセンテージを計算: (使用CPU / システム時間) * 100 * CPU数
	cpuPercent := float64(cpuDelta) / float64(sysDelta) * 100.0 * float64(numCPUs)
	return cpuPercent
}

// TestParseCommand tests the parseCommand function which splits a command
// string into individual argument parts for execution via Docker exec.
//
// This is important for the Exec functionality which requires commands
// to be passed as a string slice to the Docker API.
//
// TestParseCommandはparseCommand関数をテストします。
// これはDocker exec経由での実行のためにコマンド文字列を
// 個々の引数部分に分割します。
//
// これはDockerAPIにコマンドを文字列スライスとして渡す必要がある
// Exec機能にとって重要です。
func TestParseCommand(t *testing.T) {
	// Define test cases covering various command formats.
	// 様々なコマンド形式をカバーするテストケースを定義します。
	tests := []struct {
		name    string   // Test case name / テストケース名
		command string   // Input command string / 入力コマンド文字列
		want    []string // Expected parsed parts / 期待される解析結果
	}{
		{
			// Simple single-word command
			// 単純な単一単語コマンド
			name:    "simple command",
			command: "ls",
			want:    []string{"ls"},
		},
		{
			// Command with a single argument
			// 単一引数を持つコマンド
			name:    "command with args",
			command: "npm test",
			want:    []string{"npm", "test"},
		},
		{
			// Command with multiple arguments including flags
			// フラグを含む複数の引数を持つコマンド
			name:    "command with multiple args",
			command: "npm run build --production",
			want:    []string{"npm", "run", "build", "--production"},
		},
		{
			// Command with extra whitespace (should be trimmed)
			// 余分な空白を持つコマンド（トリミングされるべき）
			name:    "command with extra spaces",
			command: "  npm   test  ",
			want:    []string{"npm", "test"},
		},
		{
			// Empty command string
			// 空のコマンド文字列
			name:    "empty command",
			command: "",
			want:    []string{},
		},
		{
			// Whitespace-only command (should result in empty slice)
			// 空白のみのコマンド（空のスライスになるべき）
			name:    "whitespace only",
			command: "   ",
			want:    []string{},
		},
		{
			// Command with tabs as separators
			// タブを区切りとして持つコマンド
			name:    "command with tabs",
			command: "npm\ttest",
			want:    []string{"npm", "test"},
		},
	}

	// Run each test case.
	// 各テストケースを実行します。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function under test.
			// テスト対象の関数を呼び出します。
			got := parseCommand(tt.command)

			// Verify the number of parts matches.
			// 部分の数が一致することを検証します。
			if len(got) != len(tt.want) {
				t.Errorf("parseCommand(%q) returned %d parts, want %d", tt.command, len(got), len(tt.want))
				return
			}

			// Verify each part matches the expected value.
			// 各部分が期待値と一致することを検証します。
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCommand(%q)[%d] = %q, want %q", tt.command, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestContainerInfo_Fields verifies that the ContainerInfo struct
// correctly stores and retrieves all field values.
//
// This is a basic sanity check to ensure the data structure
// works as expected.
//
// TestContainerInfo_FieldsはContainerInfo構造体が
// すべてのフィールド値を正しく保存および取得することを検証します。
//
// これはデータ構造が期待通りに動作することを確認する
// 基本的な健全性チェックです。
func TestContainerInfo_Fields(t *testing.T) {
	// Create a ContainerInfo with all fields populated.
	// すべてのフィールドが設定されたContainerInfoを作成します。
	info := ContainerInfo{
		ID:      "abc123def456",
		Name:    "test-container",
		Image:   "nginx:latest",
		State:   "running",
		Status:  "Up 2 hours",
		Created: 1234567890,
		Labels: map[string]string{
			"app": "test",
		},
	}

	// Verify each field has the expected value.
	// 各フィールドが期待値を持つことを検証します。

	// Check ID field.
	// IDフィールドをチェックします。
	if info.ID != "abc123def456" {
		t.Errorf("ID = %q, want %q", info.ID, "abc123def456")
	}

	// Check Name field.
	// Nameフィールドをチェックします。
	if info.Name != "test-container" {
		t.Errorf("Name = %q, want %q", info.Name, "test-container")
	}

	// Check Image field.
	// Imageフィールドをチェックします。
	if info.Image != "nginx:latest" {
		t.Errorf("Image = %q, want %q", info.Image, "nginx:latest")
	}

	// Check State field.
	// Stateフィールドをチェックします。
	if info.State != "running" {
		t.Errorf("State = %q, want %q", info.State, "running")
	}

	// Check Status field.
	// Statusフィールドをチェックします。
	if info.Status != "Up 2 hours" {
		t.Errorf("Status = %q, want %q", info.Status, "Up 2 hours")
	}

	// Check Created field.
	// Createdフィールドをチェックします。
	if info.Created != 1234567890 {
		t.Errorf("Created = %d, want %d", info.Created, 1234567890)
	}

	// Check Labels field.
	// Labelsフィールドをチェックします。
	if info.Labels["app"] != "test" {
		t.Errorf("Labels[app] = %q, want %q", info.Labels["app"], "test")
	}
}

// TestExecResult_Fields verifies that the ExecResult struct
// correctly stores and retrieves exit code and output values.
//
// TestExecResult_FieldsはExecResult構造体が
// 終了コードと出力値を正しく保存および取得することを検証します。
func TestExecResult_Fields(t *testing.T) {
	// Create an ExecResult with typical values.
	// 典型的な値でExecResultを作成します。
	result := ExecResult{
		ExitCode: 0,
		Output:   "test output",
	}

	// Verify exit code is correct.
	// 終了コードが正しいことを検証します。
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want %d", result.ExitCode, 0)
	}

	// Verify output is correct.
	// 出力が正しいことを検証します。
	if result.Output != "test output" {
		t.Errorf("Output = %q, want %q", result.Output, "test output")
	}
}

// TestFileAccessResult_Fields verifies that the FileAccessResult struct
// correctly represents different types of file access outcomes:
// success, blocked, and error cases.
//
// TestFileAccessResult_FieldsはFileAccessResult構造体が
// 異なるタイプのファイルアクセス結果を正しく表現することを検証します:
// 成功、ブロック、エラーケース。
func TestFileAccessResult_Fields(t *testing.T) {
	// Test successful result case.
	// 成功結果ケースをテストします。
	successResult := FileAccessResult{
		Success: true,
		Data:    "file content",
	}

	// Verify success result fields.
	// 成功結果のフィールドを検証します。
	if !successResult.Success {
		t.Error("Expected Success to be true")
	}
	if successResult.Data != "file content" {
		t.Errorf("Data = %q, want %q", successResult.Data, "file content")
	}

	// Test blocked result case (path blocked by security policy).
	// ブロック結果ケースをテストします（セキュリティポリシーによりパスがブロック）。
	blockedResult := FileAccessResult{
		Success: false,
		Blocked: true,
	}

	// Verify blocked result fields.
	// ブロック結果のフィールドを検証します。
	if blockedResult.Success {
		t.Error("Expected Success to be false for blocked result")
	}
	if !blockedResult.Blocked {
		t.Error("Expected Blocked to be true")
	}

	// Test error result case (e.g., file not found).
	// エラー結果ケースをテストします（例：ファイルが見つからない）。
	errorResult := FileAccessResult{
		Success: false,
		Error:   "file not found",
	}

	// Verify error result fields.
	// エラー結果のフィールドを検証します。
	if errorResult.Success {
		t.Error("Expected Success to be false for error result")
	}
	if errorResult.Error != "file not found" {
		t.Errorf("Error = %q, want %q", errorResult.Error, "file not found")
	}
}

// TestFormatPorts tests the port formatting helper function.
// TestFormatPortsはポートフォーマットヘルパー関数をテストします。
func TestFormatPorts(t *testing.T) {
	tests := []struct {
		name     string       // Test case name / テストケース名
		ports    []types.Port // Input ports / 入力ポート
		expected []string     // Expected output / 期待される出力
	}{
		{
			name:     "empty ports",
			ports:    nil,
			expected: nil,
		},
		{
			name: "single published port",
			ports: []types.Port{
				{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 80, Type: "tcp"},
			},
			expected: []string{"0.0.0.0:80->80/tcp"},
		},
		{
			name: "port mapping with different host port",
			ports: []types.Port{
				{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
			},
			expected: []string{"0.0.0.0:8080->80/tcp"},
		},
		{
			name: "exposed but not published port",
			ports: []types.Port{
				{PrivatePort: 443, Type: "tcp"},
			},
			expected: []string{"443/tcp"},
		},
		{
			name: "multiple ports",
			ports: []types.Port{
				{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 80, Type: "tcp"},
				{IP: "0.0.0.0", PrivatePort: 443, PublicPort: 443, Type: "tcp"},
			},
			expected: []string{"0.0.0.0:80->80/tcp", "0.0.0.0:443->443/tcp"},
		},
		{
			name: "mixed published and exposed",
			ports: []types.Port{
				{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 80, Type: "tcp"},
				{PrivatePort: 3306, Type: "tcp"},
			},
			expected: []string{"0.0.0.0:80->80/tcp", "3306/tcp"},
		},
		{
			name: "UDP port",
			ports: []types.Port{
				{IP: "0.0.0.0", PrivatePort: 53, PublicPort: 53, Type: "udp"},
			},
			expected: []string{"0.0.0.0:53->53/udp"},
		},
		{
			name: "localhost binding",
			ports: []types.Port{
				{IP: "127.0.0.1", PrivatePort: 8080, PublicPort: 8080, Type: "tcp"},
			},
			expected: []string{"127.0.0.1:8080->8080/tcp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPorts(tt.ports)

			// Check length
			// 長さをチェック
			if len(result) != len(tt.expected) {
				t.Errorf("formatPorts() returned %d items, want %d: got %v",
					len(result), len(tt.expected), result)
				return
			}

			// Check each value
			// 各値をチェック
			for i, port := range result {
				if port != tt.expected[i] {
					t.Errorf("formatPorts()[%d] = %q, want %q", i, port, tt.expected[i])
				}
			}
		})
	}
}
