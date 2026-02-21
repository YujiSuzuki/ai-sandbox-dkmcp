// search-history.go - Claude Code conversation history search tool
//
// Usage:
//   go run .sandbox/tools/search-history.go [options] <pattern>
//   go run .sandbox/tools/search-history.go -list [-after DATE] [-before DATE]
//   go run .sandbox/tools/search-history.go -session <id> [-after DATE] [-before DATE]
//   go run .sandbox/tools/search-history.go -stats [-daily] [-csv|-tsv] [-after DATE] [-before DATE]
//
//   Modes:    <pattern> (keyword/regex search), -list (sessions), -session <id> (view), -stats (breakdown)
//   Filters:  -role user|assistant|tool, -tool <name>, -after/-before <YYYY-MM-DD> (local TZ), -i, -human
//   Scope:    -project <name> (default: workspace, "all" for all projects), -dir <path> (extra JSONL dir)
//   Display:  -max <n> (default: 50, 0=unlimited), -context <n> (default: 200, 0=full), -no-color
//   Output:   -csv (CSV format), -tsv (TSV format). Labels auto-detect Japanese locale (LANG).
//
//   -human filters to only actual human-typed input, excluding tool_result, IDE events, system reminders.
//   -stats shows message type breakdown. Combine with -daily for per-day data, -csv/-tsv for export.
//
//   -list shows sessions sorted by last activity. Multi-day sessions display a date range.
//   -after/-before filter by message timestamps (not session start), so they work with -list and -session too.
//
//   -dir merges an additional JSONL directory with the default ~/.claude/projects/ data.
//   Useful for combining data from multiple Sandbox environments. Export data with
//   .sandbox/host-tools/copy-credentials.sh --export. Duplicate session IDs are skipped
//   (the first occurrence, typically from ~/.claude, takes priority).
//
// Examples:
//   go run .sandbox/tools/search-history.go "DockMCP"
//   go run .sandbox/tools/search-history.go -role user "docker"
//   go run .sandbox/tools/search-history.go -role tool -tool Bash "git"
//   go run .sandbox/tools/search-history.go -after 2026-01-20 "secret"
//   go run .sandbox/tools/search-history.go -project all "error"
//   go run .sandbox/tools/search-history.go -list
//   go run .sandbox/tools/search-history.go -list -after 2026-02-08 -before 2026-02-08
//   go run .sandbox/tools/search-history.go -session c01514d6 -context 0 -max 0
//   go run .sandbox/tools/search-history.go -human -role user -max 0 ".*"
//   go run .sandbox/tools/search-history.go -stats
//   go run .sandbox/tools/search-history.go -stats -daily
//   go run .sandbox/tools/search-history.go -stats -daily -csv
//   go run .sandbox/tools/search-history.go -stats -after 2026-02-01 -daily -tsv
//   go run .sandbox/tools/search-history.go -stats -daily -dir /path/to/backup/projects/-workspace/
//
// --- 日本語 ---
//
// Claude Code の会話履歴をキーワード検索・セッション閲覧するツール。
//
// モード:
//   <pattern>        キーワード検索（正規表現対応）
//   -list            セッション一覧を表示（最終活動日順、複数日セッションは期間表示）
//   -session <id>    指定セッションを時系列で閲覧（IDは先頭数文字の前方一致）
//   -stats           メッセージ種別の内訳を表示。-daily で日別、-csv/-tsv でエクスポート
//
// フィルタ:
//   -role <role>     ロールで絞り込み: user, assistant, tool
//   -tool <name>     -role tool と併用。ツール名で絞り込み (Bash, Read, Edit, ...)
//   -after <date>    指定日以降のみ (YYYY-MM-DD, ローカルTZ)。-list, -session でも有効
//   -before <date>   指定日以前のみ (YYYY-MM-DD, ローカルTZ)。-list, -session でも有効
//   -i               大文字小文字を無視
//   -human           実際のユーザー入力のみ (tool_result, IDE イベント, system-reminder を除外)
//   -project <name>  プロジェクト指定 (デフォルト: workspace, "all" で全プロジェクト)
//   -dir <path>      追加の JSONL ディレクトリ（バックアップデータ等）
//
// 表示:
//   -max <n>         最大表示件数 (デフォルト: 50, 0 = 無制限)
//   -context <n>     1エントリの表示文字数 (デフォルト: 200, 0 = 全文)
//   -no-color        カラー出力を無効化
//   -csv             CSV形式で出力
//   -tsv             TSV形式で出力（タブ区切り）
//   ※ ラベルは LANG 環境変数が ja を含む場合、自動的に日本語になる
//
// 日付フィルタの動作:
//   -list, -session と併用時、セッション内のメッセージ日時で判定する。
//   複数日にまたがるセッションでも、指定日にメッセージがあれば表示される。
//
// -dir オプション:
//   通常の ~/.claude/projects/ に加えて、別ディレクトリの JSONL も統合して分析できる。
//   複数の Sandbox 環境のデータを横断的に集計したい場合に便利。
//   データのエクスポートには .sandbox/host-tools/copy-credentials.sh --export を使用する。
//   同一セッションID が重複する場合は、先に見つかった方（通常 ~/.claude 側）が優先される。

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// JSONL の各行の構造（必要なフィールドだけ）
type Entry struct {
	Type      string    `json:"type"`
	UserType  string    `json:"userType"`
	SessionID string    `json:"sessionId"`
	Timestamp string    `json:"timestamp"`
	IsMeta    bool      `json:"isMeta"`
	Message   *Message  `json:"message"`
	UUID      string    `json:"uuid"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type Match struct {
	SessionID   string
	SessionFile string
	Timestamp   time.Time
	Role        string
	Text        string
	MatchLine   string
}

func main() {
	var (
		roleFilter    string
		afterDate     string
		beforeDate    string
		project       string
		listSessions  bool
		maxResults    int
		contextChars  int
		ignoreCase    bool
		noColor       bool
	)

	var toolFilter string
	var sessionFilter string
	var humanOnly bool
	var showStats bool
	var daily bool
	var outputCSV bool
	var outputTSV bool
	var extraDir string

	flag.StringVar(&roleFilter, "role", "", "Filter by role: user, assistant, or tool")
	flag.StringVar(&toolFilter, "tool", "", "Filter tool_use by tool name (e.g. Bash, Read, Edit)")
	flag.StringVar(&sessionFilter, "session", "", "View a specific session (prefix match on session ID)")
	flag.StringVar(&afterDate, "after", "", "Show only after this date (YYYY-MM-DD)")
	flag.StringVar(&beforeDate, "before", "", "Show only before this date (YYYY-MM-DD)")
	flag.StringVar(&project, "project", "workspace", "Project dir name (default: workspace, use 'all' for all)")
	flag.BoolVar(&listSessions, "list", false, "List sessions with summary")
	flag.BoolVar(&showStats, "stats", false, "Show message type breakdown statistics")
	flag.BoolVar(&daily, "daily", false, "Show daily breakdown (use with -stats)")
	flag.BoolVar(&outputCSV, "csv", false, "Output in CSV format")
	flag.BoolVar(&outputTSV, "tsv", false, "Output in TSV format")
	flag.IntVar(&maxResults, "max", 50, "Max results to show (0 = unlimited)")
	flag.IntVar(&contextChars, "context", 200, "Characters of context around match / per line in session view")
	flag.BoolVar(&ignoreCase, "i", false, "Case-insensitive search")
	flag.BoolVar(&noColor, "no-color", false, "Disable color output")
	flag.BoolVar(&humanOnly, "human", false, "Only count actual human-typed input (exclude tool_result, IDE events, system reminders)")
	flag.StringVar(&extraDir, "dir", "", "Additional JSONL directory to include (e.g. backup data)")
	flag.Parse()

	if noColor {
		disableColors()
	}

	claudeDir := findClaudeDir()
	if claudeDir == "" {
		fmt.Fprintln(os.Stderr, "Error: ~/.claude/projects/ not found")
		os.Exit(1)
	}

	projectDirs := findProjectDirs(claudeDir, project)
	if extraDir != "" {
		if info, err := os.Stat(extraDir); err == nil && info.IsDir() {
			projectDirs = append(projectDirs, extraDir)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: -dir %q not found or not a directory\n", extraDir)
		}
	}
	if len(projectDirs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no project directories found for %q\n", project)
		os.Exit(1)
	}

	// Parse date filters early (shared by search and list modes)
	var afterTime, beforeTime time.Time
	var err error
	loc := time.Now().Location()
	if afterDate != "" {
		afterTime, err = time.ParseInLocation("2006-01-02", afterDate, loc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -after date: %v\n", err)
			os.Exit(1)
		}
	}
	if beforeDate != "" {
		beforeTime, err = time.ParseInLocation("2006-01-02", beforeDate, loc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -before date: %v\n", err)
			os.Exit(1)
		}
		beforeTime = beforeTime.Add(24 * time.Hour) // include the whole day
	}

	if showStats {
		runStats(projectDirs, afterTime, beforeTime, daily, outputCSV, outputTSV)
		return
	}

	if listSessions {
		listAllSessions(projectDirs, afterTime, beforeTime)
		return
	}

	if sessionFilter != "" {
		viewSession(projectDirs, sessionFilter, roleFilter, toolFilter, maxResults, contextChars, afterTime, beforeTime, humanOnly)
		return
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go run .sandbox/search-history.go [options] <pattern>")
		fmt.Fprintln(os.Stderr, "       go run .sandbox/search-history.go -list")
		fmt.Fprintln(os.Stderr, "")
		flag.PrintDefaults()
		os.Exit(1)
	}

	pattern := flag.Arg(0)
	var re *regexp.Regexp
	if ignoreCase {
		re, err = regexp.Compile("(?i)" + pattern)
	} else {
		re, err = regexp.Compile(pattern)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid pattern: %v\n", err)
		os.Exit(1)
	}

	var matches []Match
	totalFiles := 0
	totalEntries := 0
	searchSeen := make(map[string]bool)

	for _, dir := range projectDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		for _, f := range files {
			base := filepath.Base(f)
			if searchSeen[base] {
				continue
			}
			searchSeen[base] = true
			totalFiles++
			fileMatches, entries := searchFile(f, re, roleFilter, toolFilter, afterTime, beforeTime, contextChars, humanOnly)
			totalEntries += entries
			matches = append(matches, fileMatches...)
		}
	}

	// Sort by timestamp
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Timestamp.Before(matches[j].Timestamp)
	})

	if len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo matches found%s (searched %d files, %d entries)\n",
			colorDim, colorReset, totalFiles, totalEntries)
		return
	}

	// Print results
	shown := 0
	lastSession := ""
	for _, m := range matches {
		if maxResults > 0 && shown >= maxResults {
			break
		}

		if m.SessionID != lastSession {
			if lastSession != "" {
				fmt.Println()
			}
			fmt.Printf("%s── session: %s ──%s\n", colorCyan, shortID(m.SessionID), colorReset)
			lastSession = m.SessionID
		}

		roleColor := colorGreen
		if m.Role == "assistant" {
			roleColor = colorYellow
		} else if strings.HasPrefix(m.Role, "tool:") {
			roleColor = colorCyan
		}

		ts := m.Timestamp.Local().Format("01/02 15:04")
		fmt.Printf("  %s%s%s %s%-9s%s %s\n",
			colorDim, ts, colorReset,
			roleColor, m.Role, colorReset,
			highlightMatch(m.MatchLine, re))
		shown++
	}

	fmt.Printf("\n%s%d matches in %d sessions (searched %d files, %d entries)%s\n",
		colorDim, len(matches), countSessions(matches), totalFiles, totalEntries, colorReset)
	if maxResults > 0 && len(matches) > maxResults {
		fmt.Printf("%s(showing first %d, use -max 0 for unlimited)%s\n", colorDim, maxResults, colorReset)
	}
}

func viewSession(projectDirs []string, sessionPrefix, roleFilter, toolFilter string, maxResults, contextChars int, after, before time.Time, humanOnly bool) {
	// Find session file by prefix match
	var sessionPath string
	for _, dir := range projectDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		for _, f := range files {
			base := strings.TrimSuffix(filepath.Base(f), ".jsonl")
			if strings.HasPrefix(base, sessionPrefix) {
				sessionPath = f
				break
			}
		}
		if sessionPath != "" {
			break
		}
	}

	if sessionPath == "" {
		fmt.Fprintf(os.Stderr, "Session not found: %s\n", sessionPrefix)
		os.Exit(1)
	}

	f, err := os.Open(sessionPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening session: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	sessionID := strings.TrimSuffix(filepath.Base(sessionPath), ".jsonl")
	fmt.Printf("%s── session: %s ──%s\n\n", colorCyan, sessionID, colorReset)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	shown := 0
	for scanner.Scan() {
		if maxResults > 0 && shown >= maxResults {
			fmt.Printf("\n%s(showing first %d entries, use -max 0 for unlimited)%s\n", colorDim, maxResults, colorReset)
			return
		}

		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Message == nil || entry.Type == "file-history-snapshot" {
			continue
		}
		if entry.IsMeta {
			continue
		}

		role := entry.Message.Role
		if role != "user" && role != "assistant" {
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			ts, err = time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil {
				continue
			}
		}

		// Date filter
		if !after.IsZero() && ts.Before(after) {
			continue
		}
		if !before.IsZero() && ts.After(before) {
			continue
		}

		tsStr := ts.Local().Format("01/02 15:04:05")

		// Show tool_use blocks
		if role == "assistant" {
			toolTexts := extractToolUse(entry.Message.Content, toolFilter)
			for _, tt := range toolTexts {
				if roleFilter != "" && roleFilter != "tool" {
					continue
				}
				fmt.Printf("  %s%s%s %s%-12s%s %s\n",
					colorDim, tsStr, colorReset,
					colorCyan, "tool:"+tt.name, colorReset,
					truncateView(tt.text, contextChars))
				shown++
			}
		}

		if roleFilter == "tool" {
			continue
		}
		if roleFilter != "" && role != roleFilter {
			continue
		}

		// Human-only filter: skip user messages that are not actual human input
		if humanOnly && role == "user" && !isHumanInput(entry.Message.Content) {
			continue
		}

		text := extractText(entry.Message.Content)
		if text == "" {
			continue
		}

		// Collapse whitespace
		text = strings.Join(strings.Fields(text), " ")

		roleColor := colorGreen
		if role == "assistant" {
			roleColor = colorYellow
		}

		// Truncate long messages for overview
		display := truncateView(text, contextChars)

		fmt.Printf("  %s%s%s %s%-12s%s %s\n",
			colorDim, tsStr, colorReset,
			roleColor, role, colorReset,
			display)
		shown++
	}

}

func truncateView(s string, n int) string {
	if n <= 0 {
		return s
	}
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func findClaudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".claude", "projects")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return ""
}

func findProjectDirs(claudeDir, project string) []string {
	if project == "all" {
		entries, _ := os.ReadDir(claudeDir)
		var dirs []string
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(claudeDir, e.Name()))
			}
		}
		return dirs
	}

	// Try exact match first, then with dash prefix
	candidates := []string{
		filepath.Join(claudeDir, project),
		filepath.Join(claudeDir, "-"+project),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return []string{c}
		}
	}

	// Glob match
	entries, _ := os.ReadDir(claudeDir)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && strings.Contains(e.Name(), project) {
			dirs = append(dirs, filepath.Join(claudeDir, e.Name()))
		}
	}
	return dirs
}

func searchFile(path string, re *regexp.Regexp, roleFilter, toolFilter string, after, before time.Time, contextChars int, humanOnly bool) ([]Match, int) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0
	}
	defer f.Close()

	sessionFile := filepath.Base(path)
	var matches []Match
	entries := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // up to 10MB per line

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// Skip non-message entries
		if entry.Message == nil || entry.Type == "file-history-snapshot" {
			continue
		}

		// Skip meta messages (commands, system)
		if entry.IsMeta {
			continue
		}

		role := entry.Message.Role
		if role != "user" && role != "assistant" {
			continue
		}

		// Time filter
		ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			ts, err = time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil {
				continue
			}
		}
		if !after.IsZero() && ts.Before(after) {
			continue
		}
		if !before.IsZero() && ts.After(before) {
			continue
		}

		sessionID := entry.SessionID
		if sessionID == "" {
			sessionID = strings.TrimSuffix(sessionFile, ".jsonl")
		}

		// Extract tool_use blocks from assistant messages
		if role == "assistant" {
			toolTexts := extractToolUse(entry.Message.Content, toolFilter)
			for _, tt := range toolTexts {
				entries++
				if roleFilter != "" && roleFilter != "tool" {
					continue
				}
				loc := re.FindStringIndex(tt.text)
				if loc == nil {
					continue
				}
				matchLine := extractContext(tt.text, loc, contextChars)
				matches = append(matches, Match{
					SessionID:   sessionID,
					SessionFile: sessionFile,
					Timestamp:   ts,
					Role:        "tool:" + tt.name,
					Text:        tt.text,
					MatchLine:   matchLine,
				})
			}
		}

		entries++

		// Role filter: "tool" only matches tool_use, skip text
		if roleFilter == "tool" {
			continue
		}
		if roleFilter != "" && role != roleFilter {
			continue
		}

		// Human-only filter: skip user messages that are not actual human input
		if humanOnly && role == "user" && !isHumanInput(entry.Message.Content) {
			continue
		}

		// Extract text content
		text := extractText(entry.Message.Content)
		if text == "" {
			continue
		}

		// Search
		loc := re.FindStringIndex(text)
		if loc == nil {
			continue
		}

		// Extract context around match
		matchLine := extractContext(text, loc, contextChars)

		matches = append(matches, Match{
			SessionID:   sessionID,
			SessionFile: sessionFile,
			Timestamp:   ts,
			Role:        role,
			Text:        text,
			MatchLine:   matchLine,
		})
	}

	return matches, entries
}

type toolUseText struct {
	name string
	text string
}

func extractToolUse(content interface{}, toolFilter string) []toolUseText {
	arr, ok := content.([]interface{})
	if !ok {
		return nil
	}

	var results []toolUseText
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := m["type"].(string); t != "tool_use" {
			continue
		}

		name, _ := m["name"].(string)
		if toolFilter != "" && !strings.EqualFold(name, toolFilter) {
			continue
		}

		input, _ := m["input"].(map[string]interface{})
		if input == nil {
			continue
		}

		// Build searchable text from tool name + input fields
		var parts []string
		parts = append(parts, "["+name+"]")

		// Prioritize command/description for Bash
		if cmd, ok := input["command"].(string); ok {
			parts = append(parts, cmd)
		}
		if desc, ok := input["description"].(string); ok {
			parts = append(parts, "("+desc+")")
		}
		// file_path for Read/Edit/Write
		if fp, ok := input["file_path"].(string); ok {
			parts = append(parts, fp)
		}
		// old_string/new_string for Edit
		if old, ok := input["old_string"].(string); ok {
			parts = append(parts, "old:"+truncate(old, 80))
		}
		if ns, ok := input["new_string"].(string); ok {
			parts = append(parts, "new:"+truncate(ns, 80))
		}
		// pattern for Grep/Glob
		if pat, ok := input["pattern"].(string); ok {
			parts = append(parts, "pattern:"+pat)
		}
		// prompt for Task/WebFetch
		if prompt, ok := input["prompt"].(string); ok {
			parts = append(parts, "prompt:"+truncate(prompt, 120))
		}

		results = append(results, toolUseText{name: name, text: strings.Join(parts, " ")})
	}
	return results
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// isHumanInput checks if a user message contains actual human-typed text
// (not just tool_result, IDE events, or system-injected content).
func isHumanInput(content interface{}) bool {
	switch v := content.(type) {
	case string:
		return v != "" && !isAutomatedText(v)
	case []interface{}:
		for _, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			t, _ := m["type"].(string)
			if t != "text" {
				continue
			}
			text, _ := m["text"].(string)
			if text == "" {
				continue
			}
			if !isAutomatedText(text) {
				return true
			}
		}
	}
	return false
}

// isAutomatedText returns true if the text block is auto-generated (not typed by user).
func isAutomatedText(text string) bool {
	trimmed := strings.TrimSpace(text)
	prefixes := []string{
		"<ide_opened_file>",
		"<ide_selection>",
		"<system-reminder>",
		"<user-prompt-submit-hook>",
		"<command-name>",
		"<command-message>",
		"[Request interrupted by user",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func extractText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func extractContext(text string, loc []int, contextChars int) string {
	// Collapse whitespace for display
	text = strings.Join(strings.Fields(text), " ")

	// Re-find in collapsed text
	// (position may shift, so just take a window from the start of match)
	start := 0
	end := len(text)

	if len(text) <= contextChars*2 {
		return text
	}

	// Find match position in collapsed text (approximate)
	matchStart := loc[0]
	if matchStart > len(text) {
		matchStart = 0
	}

	start = matchStart - contextChars
	if start < 0 {
		start = 0
	}
	end = matchStart + contextChars
	if end > len(text) {
		end = len(text)
	}

	result := text[start:end]
	if start > 0 {
		result = "..." + result
	}
	if end < len(text) {
		result = result + "..."
	}
	return result
}

func highlightMatch(text string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(text, func(s string) string {
		return colorRed + colorBold + s + colorReset
	})
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func countSessions(matches []Match) int {
	seen := make(map[string]bool)
	for _, m := range matches {
		seen[m.SessionID] = true
	}
	return len(seen)
}

func listAllSessions(projectDirs []string, after, before time.Time) {
	type SessionInfo struct {
		ID        string
		Project   string
		FirstTime time.Time
		LastTime  time.Time
		Messages  int
		FirstMsg  string
		FileSize  int64
	}

	var sessions []SessionInfo
	listSeen := make(map[string]bool)

	for _, dir := range projectDirs {
		project := filepath.Base(dir)
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))

		for _, path := range files {
			if base := filepath.Base(path); listSeen[base] {
				continue
			} else {
				listSeen[base] = true
			}
			info, err := os.Stat(path)
			if err != nil || info.Size() == 0 {
				continue
			}

			f, err := os.Open(path)
			if err != nil {
				continue
			}

			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

			si := SessionInfo{
				ID:       strings.TrimSuffix(filepath.Base(path), ".jsonl"),
				Project:  project,
				FileSize: info.Size(),
			}

			hasMatchingMsg := false
			for scanner.Scan() {
				var entry Entry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					continue
				}
				if entry.Message == nil || entry.IsMeta {
					continue
				}
				if entry.Message.Role != "user" && entry.Message.Role != "assistant" {
					continue
				}

				ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
				if err != nil {
					ts, err = time.Parse(time.RFC3339, entry.Timestamp)
					if err != nil {
						continue
					}
				}

				si.Messages++
				if si.FirstTime.IsZero() || ts.Before(si.FirstTime) {
					si.FirstTime = ts
				}
				if ts.After(si.LastTime) {
					si.LastTime = ts
				}

				// Check if any message falls within the date filter range
				if !after.IsZero() || !before.IsZero() {
					inRange := true
					if !after.IsZero() && ts.Before(after) {
						inRange = false
					}
					if !before.IsZero() && ts.After(before) {
						inRange = false
					}
					if inRange {
						hasMatchingMsg = true
					}
				}

				// Capture first user message as summary
				if si.FirstMsg == "" && entry.Message.Role == "user" {
					text := extractText(entry.Message.Content)
					text = strings.Join(strings.Fields(text), " ")
					if len(text) > 80 {
						text = text[:80] + "..."
					}
					si.FirstMsg = text
				}
			}
			f.Close()

			// Skip sessions with no messages
			if si.Messages == 0 {
				continue
			}
			// If date filter is active, skip sessions without matching messages
			if (!after.IsZero() || !before.IsZero()) && !hasMatchingMsg {
				continue
			}
			sessions = append(sessions, si)
		}
	}

	// Sort by last activity (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastTime.After(sessions[j].LastTime)
	})

	for _, s := range sessions {
		dateStr := formatDateRange(s.FirstTime, s.LastTime)
		sizeKB := s.FileSize / 1024
		fmt.Printf("%s%s%s  %s%-20s%s  %s%4d msgs%s  %s%4dKB%s  %s\n",
			colorCyan, shortID(s.ID), colorReset,
			colorDim, dateStr, colorReset,
			colorGreen, s.Messages, colorReset,
			colorDim, sizeKB, colorReset,
			s.FirstMsg)
	}
	fmt.Printf("\n%s%d sessions%s\n", colorDim, len(sessions), colorReset)
}

// formatDateRange returns a compact date range string.
// Same day:  "02/09 14:32"
// Multi-day: "02/04 14:32 ~ 02/09"
func formatDateRange(first, last time.Time) string {
	f := first.Local()
	l := last.Local()
	if f.Year() == l.Year() && f.YearDay() == l.YearDay() {
		return f.Format("01/02 15:04")
	}
	return f.Format("01/02 15:04") + " ~ " + l.Format("01/02")
}

// --- stats mode ---

type statsCount struct {
	HumanInput    int
	Interrupted   int
	SlashCommands int
	IDEEvents     int
	SystemMsgs    int
	ToolResults   int
	AssistantMsgs int
	Sessions      int
}

func (s *statsCount) TotalUser() int {
	return s.HumanInput + s.Interrupted + s.SlashCommands + s.IDEEvents + s.SystemMsgs + s.ToolResults
}

func (s *statsCount) Total() int {
	return s.TotalUser() + s.AssistantMsgs
}

func isJapaneseLocale() bool {
	for _, key := range []string{"LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"} {
		if v := os.Getenv(key); strings.Contains(strings.ToLower(v), "ja") {
			return true
		}
	}
	return false
}

// classifyUserMessage categorizes a user message content into one type.
// classifyUserMessage はユーザーメッセージを1つのカテゴリに分類する。
//
// Returns / 戻り値: "human", "interrupt", "command", "ide", "system", "tool_result", "automated", "empty"
func classifyUserMessage(content interface{}) string {
	arr, ok := content.([]interface{})
	if !ok {
		if s, ok := content.(string); ok && s != "" {
			if isAutomatedText(s) {
				return "automated"
			}
			return "human"
		}
		return "empty"
	}

	hasHuman := false
	hasToolResult := false
	hasIDE := false
	hasCommand := false
	hasSystem := false
	hasInterrupt := false

	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := m["type"].(string)

		if t == "tool_result" {
			hasToolResult = true
			continue
		}
		if t != "text" {
			continue
		}

		text, _ := m["text"].(string)
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "<ide_opened_file>") || strings.HasPrefix(trimmed, "<ide_selection>") {
			hasIDE = true
		} else if strings.HasPrefix(trimmed, "<system-reminder>") || strings.HasPrefix(trimmed, "<user-prompt-submit-hook>") {
			hasSystem = true
		} else if strings.HasPrefix(trimmed, "<command-name>") || strings.HasPrefix(trimmed, "<command-message>") {
			hasCommand = true
		} else if strings.HasPrefix(trimmed, "[Request interrupted by user") {
			hasInterrupt = true
		} else {
			hasHuman = true
		}
	}

	// Priority: if a message contains multiple content types, the most
	// "intentional" one wins. Human input takes precedence because the user
	// actively typed something, regardless of accompanying tool_result or
	// system blocks injected by the framework.
	// 優先順位: 複数のコンテンツが混在する場合、最も「意図的」なものを採用する。
	// ユーザーが実際に入力したテキストがあれば、フレームワークが付加した
	// tool_result や system ブロックより常に優先される。
	if hasHuman {
		return "human"
	}
	if hasInterrupt {
		return "interrupt"
	}
	if hasCommand {
		return "command"
	}
	if hasIDE {
		return "ide"
	}
	if hasSystem {
		return "system"
	}
	if hasToolResult {
		return "tool_result"
	}
	return "empty"
}

func runStats(projectDirs []string, after, before time.Time, daily, csvOut, tsvOut bool) {
	type dayKey = string
	totals := statsCount{}
	dailyMap := make(map[dayKey]*statsCount)
	sessionsSeen := make(map[string]bool)
	dailySessions := make(map[dayKey]map[string]bool)

	filesSeen := make(map[string]bool)
	for _, dir := range projectDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		for _, path := range files {
			sessionFile := filepath.Base(path)
			if filesSeen[sessionFile] {
				continue // skip duplicate (backup vs current)
			}
			filesSeen[sessionFile] = true

			f, err := os.Open(path)
			if err != nil {
				continue
			}

			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

			for scanner.Scan() {
				var entry Entry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					continue
				}
				if entry.Message == nil || entry.Type == "file-history-snapshot" || entry.IsMeta {
					continue
				}

				role := entry.Message.Role
				if role != "user" && role != "assistant" {
					continue
				}

				ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
				if err != nil {
					ts, err = time.Parse(time.RFC3339, entry.Timestamp)
					if err != nil {
						continue
					}
				}
				if !after.IsZero() && ts.Before(after) {
					continue
				}
				if !before.IsZero() && ts.After(before) {
					continue
				}

				sessionID := entry.SessionID
				if sessionID == "" {
					sessionID = strings.TrimSuffix(sessionFile, ".jsonl")
				}

				dk := ts.Local().Format("2006-01-02")
				if _, ok := dailyMap[dk]; !ok {
					dailyMap[dk] = &statsCount{}
					dailySessions[dk] = make(map[string]bool)
				}

				if !sessionsSeen[sessionID] {
					sessionsSeen[sessionID] = true
					totals.Sessions++
				}
				if !dailySessions[dk][sessionID] {
					dailySessions[dk][sessionID] = true
					dailyMap[dk].Sessions++
				}

				if role == "assistant" {
					totals.AssistantMsgs++
					dailyMap[dk].AssistantMsgs++
					continue
				}

				// role == "user"
				cat := classifyUserMessage(entry.Message.Content)
				switch cat {
				case "human":
					totals.HumanInput++
					dailyMap[dk].HumanInput++
				case "interrupt":
					totals.Interrupted++
					dailyMap[dk].Interrupted++
				case "command":
					totals.SlashCommands++
					dailyMap[dk].SlashCommands++
				case "ide":
					totals.IDEEvents++
					dailyMap[dk].IDEEvents++
				case "system":
					totals.SystemMsgs++
					dailyMap[dk].SystemMsgs++
				case "tool_result":
					totals.ToolResults++
					dailyMap[dk].ToolResults++
				case "automated":
					totals.SystemMsgs++
					dailyMap[dk].SystemMsgs++
				}
			}
			f.Close()
		}
	}

	ja := isJapaneseLocale()

	if csvOut || tsvOut {
		printStatsDelimited(totals, dailyMap, daily, csvOut, ja)
	} else {
		printStatsTable(totals, dailyMap, daily, ja)
	}
}

type statsLabel struct {
	key   string
	en    string
	ja    string
	value func(*statsCount) int
}

func statsLabels() []statsLabel {
	return []statsLabel{
		{"human_input", "Human input", "ユーザー入力", func(s *statsCount) int { return s.HumanInput }},
		{"interrupted", "Interrupted", "中断", func(s *statsCount) int { return s.Interrupted }},
		{"slash_commands", "Slash commands", "スラッシュコマンド", func(s *statsCount) int { return s.SlashCommands }},
		{"ide_events", "IDE events", "IDE イベント", func(s *statsCount) int { return s.IDEEvents }},
		{"system_msgs", "System messages", "システムメッセージ", func(s *statsCount) int { return s.SystemMsgs }},
		{"tool_results", "tool_result", "tool_result", func(s *statsCount) int { return s.ToolResults }},
		{"assistant_msgs", "Assistant", "アシスタント", func(s *statsCount) int { return s.AssistantMsgs }},
		{"sessions", "Sessions", "セッション数", func(s *statsCount) int { return s.Sessions }},
		{"total", "Total", "合計", func(s *statsCount) int { return s.Total() }},
	}
}

func printStatsTable(totals statsCount, dailyMap map[string]*statsCount, daily, ja bool) {
	labels := statsLabels()

	// Summary
	headerLabel := "Message Breakdown"
	avgLabel := "Per Day"
	if ja {
		headerLabel = "メッセージ内訳"
		avgLabel = "1日平均"
	}

	days := len(dailyMap)
	if days == 0 {
		days = 1
	}

	fmt.Printf("\n%s%s── %s ──%s\n\n", colorBold, colorCyan, headerLabel, colorReset)

	// Find max label width for alignment
	maxLabelW := 0
	for _, l := range labels {
		name := l.en
		if ja {
			name = l.ja
		}
		if len(name) > maxLabelW {
			maxLabelW = len(name)
		}
	}
	// Japanese strings are wider in display; add padding
	if ja {
		maxLabelW += 8
	}

	for i, l := range labels {
		name := l.en
		if ja {
			name = l.ja
		}
		val := l.value(&totals)
		avg := float64(val) / float64(days)

		if i == len(labels)-1 { // before "Total"
			fmt.Printf("  %s%s%s\n", colorDim, strings.Repeat("─", maxLabelW+20), colorReset)
		}

		color := colorReset
		if l.key == "human_input" {
			color = colorGreen
		} else if l.key == "total" {
			color = colorBold
		}

		fmt.Printf("  %s%-*s%s  %s%6d%s  %s(%s: %.1f)%s\n",
			color, maxLabelW, name, colorReset,
			color, val, colorReset,
			colorDim, avgLabel, avg, colorReset)
	}

	fmt.Printf("\n  %s%d %s, %d %s%s\n",
		colorDim, days,
		map[bool]string{true: "日間", false: "days"}[ja],
		totals.Sessions,
		map[bool]string{true: "セッション", false: "sessions"}[ja],
		colorReset)

	// Daily breakdown
	if daily {
		printDailyTable(dailyMap, ja)
	}
}

func printDailyTable(dailyMap map[string]*statsCount, ja bool) {
	// Sort dates
	dates := make([]string, 0, len(dailyMap))
	for d := range dailyMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	headerLabel := "Daily Breakdown"
	if ja {
		headerLabel = "日別内訳"
	}

	fmt.Printf("\n%s%s── %s ──%s\n\n", colorBold, colorCyan, headerLabel, colorReset)

	// Header
	var headers [8]string
	if ja {
		headers = [8]string{"日付", "入力", "中断", "コマンド", "IDE", "Sys", "Tool", "Asst"}
	} else {
		headers = [8]string{"Date", "Human", "Intrpt", "Cmd", "IDE", "Sys", "Tool", "Asst"}
	}
	fmt.Printf("  %s%-12s %6s %6s %6s %6s %6s %6s %6s%s\n",
		colorDim, headers[0], headers[1], headers[2], headers[3], headers[4], headers[5], headers[6], headers[7], colorReset)
	fmt.Printf("  %s%s%s\n", colorDim, strings.Repeat("─", 68), colorReset)

	for _, d := range dates {
		s := dailyMap[d]
		// Short date display
		t, _ := time.Parse("2006-01-02", d)
		dateStr := t.Format("01/02 (Mon)")

		fmt.Printf("  %-12s %s%6d%s %6d %6d %6d %6d %6d %6d\n",
			dateStr,
			colorGreen, s.HumanInput, colorReset,
			s.Interrupted, s.SlashCommands, s.IDEEvents,
			s.SystemMsgs, s.ToolResults, s.AssistantMsgs)
	}
}

func printStatsDelimited(totals statsCount, dailyMap map[string]*statsCount, daily, csvOut, ja bool) {
	sep := ","
	if !csvOut {
		sep = "\t"
	}

	if daily {
		// Daily CSV/TSV
		dateH := "date"
		headers := []string{dateH, "human_input", "interrupted", "slash_commands", "ide_events", "system_msgs", "tool_results", "assistant", "sessions"}
		if ja {
			headers = []string{"日付", "ユーザー入力", "中断", "スラッシュコマンド", "IDEイベント", "システム", "tool_result", "アシスタント", "セッション"}
		}
		fmt.Println(strings.Join(headers, sep))

		dates := make([]string, 0, len(dailyMap))
		for d := range dailyMap {
			dates = append(dates, d)
		}
		sort.Strings(dates)

		for _, d := range dates {
			s := dailyMap[d]
			fmt.Printf("%s%s%d%s%d%s%d%s%d%s%d%s%d%s%d%s%d\n",
				d, sep, s.HumanInput, sep, s.Interrupted, sep, s.SlashCommands, sep,
				s.IDEEvents, sep, s.SystemMsgs, sep, s.ToolResults, sep,
				s.AssistantMsgs, sep, s.Sessions)
		}
	} else {
		// Summary CSV/TSV
		catH, countH := "category", "count"
		if ja {
			catH, countH = "カテゴリ", "件数"
		}
		fmt.Printf("%s%s%s\n", catH, sep, countH)

		labels := statsLabels()
		for _, l := range labels {
			name := l.en
			if ja {
				name = l.ja
			}
			fmt.Printf("%s%s%d\n", name, sep, l.value(&totals))
		}
	}
}

func disableColors() {
	colorReset = ""
	colorDim = ""
	colorBold = ""
	colorRed = ""
	colorGreen = ""
	colorYellow = ""
	colorCyan = ""
}
