// search-history.go - Claude Code 会話履歴検索ツール (おまけツール)
//
// Usage:
//   go run .sandbox/tools/search-history.go [options] <pattern>
//
// Examples:
//   go run .sandbox/tools/search-history.go "DockMCP"
//   go run .sandbox/tools/search-history.go -role user "docker"
//   go run .sandbox/tools/search-history.go -role tool "npm test"
//   go run .sandbox/tools/search-history.go -role tool -tool Bash "git"
//   go run .sandbox/tools/search-history.go -after 2026-01-20 "secret"
//   go run .sandbox/tools/search-history.go -project all "error"
//   go run .sandbox/tools/search-history.go -list
//   go run .sandbox/tools/search-history.go -session c01514d6
//   go run .sandbox/tools/search-history.go -session c01514d6 -role tool
//
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

	flag.StringVar(&roleFilter, "role", "", "Filter by role: user, assistant, or tool")
	flag.StringVar(&toolFilter, "tool", "", "Filter tool_use by tool name (e.g. Bash, Read, Edit)")
	flag.StringVar(&sessionFilter, "session", "", "View a specific session (prefix match on session ID)")
	flag.StringVar(&afterDate, "after", "", "Show only after this date (YYYY-MM-DD)")
	flag.StringVar(&beforeDate, "before", "", "Show only before this date (YYYY-MM-DD)")
	flag.StringVar(&project, "project", "workspace", "Project dir name (default: workspace, use 'all' for all)")
	flag.BoolVar(&listSessions, "list", false, "List sessions with summary")
	flag.IntVar(&maxResults, "max", 50, "Max results to show (0 = unlimited)")
	flag.IntVar(&contextChars, "context", 200, "Characters of context around match / per line in session view")
	flag.BoolVar(&ignoreCase, "i", false, "Case-insensitive search")
	flag.BoolVar(&noColor, "no-color", false, "Disable color output")
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
	if len(projectDirs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no project directories found for %q\n", project)
		os.Exit(1)
	}

	if listSessions {
		listAllSessions(projectDirs)
		return
	}

	if sessionFilter != "" {
		viewSession(projectDirs, sessionFilter, roleFilter, toolFilter, maxResults, contextChars)
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
	var err error
	if ignoreCase {
		re, err = regexp.Compile("(?i)" + pattern)
	} else {
		re, err = regexp.Compile(pattern)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid pattern: %v\n", err)
		os.Exit(1)
	}

	var afterTime, beforeTime time.Time
	if afterDate != "" {
		afterTime, err = time.Parse("2006-01-02", afterDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -after date: %v\n", err)
			os.Exit(1)
		}
	}
	if beforeDate != "" {
		beforeTime, err = time.Parse("2006-01-02", beforeDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -before date: %v\n", err)
			os.Exit(1)
		}
		beforeTime = beforeTime.Add(24 * time.Hour) // include the whole day
	}

	var matches []Match
	totalFiles := 0
	totalEntries := 0

	for _, dir := range projectDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		for _, f := range files {
			totalFiles++
			fileMatches, entries := searchFile(f, re, roleFilter, toolFilter, afterTime, beforeTime, contextChars)
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

func viewSession(projectDirs []string, sessionPrefix, roleFilter, toolFilter string, maxResults, contextChars int) {
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
			ts, _ = time.Parse(time.RFC3339, entry.Timestamp)
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

func searchFile(path string, re *regexp.Regexp, roleFilter, toolFilter string, after, before time.Time, contextChars int) ([]Match, int) {
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

func listAllSessions(projectDirs []string) {
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

	for _, dir := range projectDirs {
		project := filepath.Base(dir)
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))

		for _, path := range files {
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
					ts, _ = time.Parse(time.RFC3339, entry.Timestamp)
				}

				si.Messages++
				if si.FirstTime.IsZero() || ts.Before(si.FirstTime) {
					si.FirstTime = ts
				}
				if ts.After(si.LastTime) {
					si.LastTime = ts
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

			if si.Messages > 0 {
				sessions = append(sessions, si)
			}
		}
	}

	// Sort by time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastTime.After(sessions[j].LastTime)
	})

	for _, s := range sessions {
		date := s.FirstTime.Local().Format("2006-01-02 15:04")
		sizeKB := s.FileSize / 1024
		fmt.Printf("%s%s%s  %s%-20s%s  %s%3d msgs%s  %s%4dKB%s  %s\n",
			colorCyan, shortID(s.ID), colorReset,
			colorDim, date, colorReset,
			colorGreen, s.Messages, colorReset,
			colorDim, sizeKB, colorReset,
			s.FirstMsg)
	}
	fmt.Printf("\n%s%d sessions%s\n", colorDim, len(sessions), colorReset)
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
