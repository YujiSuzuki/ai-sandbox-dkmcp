// current_test.go contains unit tests for the current container utilities.
// These tests verify the read, write, and clear operations for current container state.
//
// current_test.goはカレントコンテナユーティリティのユニットテストを含みます。
// これらのテストはカレントコンテナ状態の読み取り、書き込み、クリア操作を検証します。
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetCurrentContainerDir verifies that the state directory path is correctly formed.
//
// TestGetCurrentContainerDirは状態ディレクトリパスが正しく形成されることを検証します。
func TestGetCurrentContainerDir(t *testing.T) {
	dir, err := GetCurrentContainerDir()
	if err != nil {
		t.Fatalf("GetCurrentContainerDir() error: %v", err)
	}

	// Should end with .dkmcp
	// .dkmcpで終わるべき
	if filepath.Base(dir) != ".dkmcp" {
		t.Errorf("Expected directory to end with .dkmcp, got %s", dir)
	}
}

// TestGetCurrentContainerPath verifies that the state file path is correctly formed.
//
// TestGetCurrentContainerPathは状態ファイルパスが正しく形成されることを検証します。
func TestGetCurrentContainerPath(t *testing.T) {
	path, err := GetCurrentContainerPath()
	if err != nil {
		t.Fatalf("GetCurrentContainerPath() error: %v", err)
	}

	// Should end with .dkmcp/current
	// .dkmcp/currentで終わるべき
	if filepath.Base(path) != CurrentContainerStateFile {
		t.Errorf("Expected path to end with %s, got %s", CurrentContainerStateFile, filepath.Base(path))
	}
}

// TestWriteAndReadCurrentContainer verifies write and read operations work correctly.
//
// TestWriteAndReadCurrentContainerは書き込みと読み取り操作が正しく動作することを検証します。
func TestWriteAndReadCurrentContainer(t *testing.T) {
	// Use a temporary directory for testing
	// テスト用に一時ディレクトリを使用
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	testContainer := "test-container-123"

	// Write current container
	// カレントコンテナを書き込み
	err := WriteCurrentContainer(testContainer)
	if err != nil {
		t.Fatalf("WriteCurrentContainer() error: %v", err)
	}

	// Read it back
	// 読み取り
	container, err := ReadCurrentContainer()
	if err != nil {
		t.Fatalf("ReadCurrentContainer() error: %v", err)
	}

	if container != testContainer {
		t.Errorf("Expected %s, got %s", testContainer, container)
	}
}

// TestReadCurrentContainer_NotExists verifies reading returns empty when no file exists.
//
// TestReadCurrentContainer_NotExistsはファイルが存在しない場合に空が返されることを検証します。
func TestReadCurrentContainer_NotExists(t *testing.T) {
	// Use a temporary directory with no state file
	// 状態ファイルのない一時ディレクトリを使用
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	container, err := ReadCurrentContainer()
	if err != nil {
		t.Fatalf("ReadCurrentContainer() error: %v", err)
	}

	if container != "" {
		t.Errorf("Expected empty string, got %s", container)
	}
}

// TestClearCurrentContainer verifies that clear removes the state file.
//
// TestClearCurrentContainerはクリアが状態ファイルを削除することを検証します。
func TestClearCurrentContainer(t *testing.T) {
	// Use a temporary directory
	// 一時ディレクトリを使用
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Write then clear
	// 書き込んでからクリア
	err := WriteCurrentContainer("test-container")
	if err != nil {
		t.Fatalf("WriteCurrentContainer() error: %v", err)
	}

	err = ClearCurrentContainer()
	if err != nil {
		t.Fatalf("ClearCurrentContainer() error: %v", err)
	}

	// Verify it's cleared
	// クリアされたことを確認
	container, err := ReadCurrentContainer()
	if err != nil {
		t.Fatalf("ReadCurrentContainer() error: %v", err)
	}

	if container != "" {
		t.Errorf("Expected empty string after clear, got %s", container)
	}
}

// TestClearCurrentContainer_NotExists verifies that clear succeeds even when no file exists.
//
// TestClearCurrentContainer_NotExistsはファイルが存在しない場合でもクリアが成功することを検証します。
func TestClearCurrentContainer_NotExists(t *testing.T) {
	// Use a temporary directory with no state file
	// 状態ファイルのない一時ディレクトリを使用
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Clear should not return an error
	// クリアはエラーを返さないべき
	err := ClearCurrentContainer()
	if err != nil {
		t.Errorf("ClearCurrentContainer() should not error when file doesn't exist: %v", err)
	}
}

// TestWriteCurrentContainer_CreatesDirectory verifies that write creates the directory if needed.
//
// TestWriteCurrentContainer_CreatesDirectoryは必要に応じてディレクトリが作成されることを検証します。
func TestWriteCurrentContainer_CreatesDirectory(t *testing.T) {
	// Use a temporary directory
	// 一時ディレクトリを使用
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Verify .dkmcp doesn't exist yet
	// .dkmcpがまだ存在しないことを確認
	dkmcpDir := filepath.Join(tempDir, ".dkmcp")
	if _, err := os.Stat(dkmcpDir); !os.IsNotExist(err) {
		t.Fatal(".dkmcp directory should not exist initially")
	}

	// Write should create the directory
	// 書き込みがディレクトリを作成するべき
	err := WriteCurrentContainer("test-container")
	if err != nil {
		t.Fatalf("WriteCurrentContainer() error: %v", err)
	}

	// Verify directory was created
	// ディレクトリが作成されたことを確認
	if _, err := os.Stat(dkmcpDir); os.IsNotExist(err) {
		t.Error(".dkmcp directory should have been created")
	}
}

// TestGetContainerOrCurrent_WithExplicitContainer verifies explicit container is returned as-is.
//
// TestGetContainerOrCurrent_WithExplicitContainerは明示的なコンテナがそのまま返されることを検証します。
func TestGetContainerOrCurrent_WithExplicitContainer(t *testing.T) {
	explicit := "explicit-container"
	result, err := GetContainerOrCurrent(explicit)
	if err != nil {
		t.Fatalf("GetContainerOrCurrent() error: %v", err)
	}

	if result != explicit {
		t.Errorf("Expected %s, got %s", explicit, result)
	}
}

// TestWriteCurrentContainer_Whitespace verifies that whitespace is handled correctly.
// WriteCurrentContainer doesn't trim, but ReadCurrentContainer trims the value.
//
// TestWriteCurrentContainer_Whitespaceは空白が正しく処理されることを検証します。
// WriteCurrentContainerはトリムしませんが、ReadCurrentContainerは値をトリムします。
func TestWriteCurrentContainer_Whitespace(t *testing.T) {
	// Use a temporary directory
	// 一時ディレクトリを使用
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Write container name with trailing whitespace to test trimming on read
	// 読み取り時のトリミングをテストするため、末尾に空白のあるコンテナ名を書き込み
	containerWithWhitespace := "test-container  \n"
	expectedContainer := "test-container"

	err := WriteCurrentContainer(containerWithWhitespace)
	if err != nil {
		t.Fatalf("WriteCurrentContainer() error: %v", err)
	}

	// Read should return trimmed value (without trailing spaces and newline)
	// 読み取りはトリムされた値を返すべき（末尾のスペースと改行なし）
	container, err := ReadCurrentContainer()
	if err != nil {
		t.Fatalf("ReadCurrentContainer() error: %v", err)
	}

	if container != expectedContainer {
		t.Errorf("ReadCurrentContainer() = %q, want %q (should trim whitespace)", container, expectedContainer)
	}
}
