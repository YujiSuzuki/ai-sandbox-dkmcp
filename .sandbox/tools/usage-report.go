// Token usage report for Claude Code conversation history
//
// Aggregates token usage from conversation history JSONL files.
// Prices are NOT hardcoded. Use -json output and ask AI to
// fetch current pricing from docs.anthropic.com for cost estimation.
//
// Usage:
//   go run .sandbox/tools/usage-report.go [options]
//   -month             Last 30 days (default: last 7 days)
//   -after YYYY-MM-DD  From date (local timezone)
//   -before YYYY-MM-DD To date (local timezone)
//   -daily             Show daily breakdown
//   -model <name>      Filter by model (partial match, e.g. "opus", "sonnet")
//   -project <name>    Project (default: workspace, "all" for all)
//   -json              JSON output (for AI cost calculation)
//   -no-color          Disable color output
//
// Examples:
//   go run .sandbox/tools/usage-report.go
//   go run .sandbox/tools/usage-report.go -month
//   go run .sandbox/tools/usage-report.go -daily -after 2026-02-01
//   go run .sandbox/tools/usage-report.go -json -month
//   go run .sandbox/tools/usage-report.go -model opus -daily
//
// ---
//
// Claude Code の会話履歴からトークン使用量を集計するツール。
// モデル別・期間別のトークン数をレポートする。
// 料金はハードコードしていない。コスト見積もりが必要な場合は
// -json で出力し、AI に公式サイトから最新価格を取得してもらう。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// JSONL entry (only fields we need)
type Entry struct {
	Type      string   `json:"type"`
	SessionID string   `json:"sessionId"`
	Timestamp string   `json:"timestamp"`
	IsMeta    bool     `json:"isMeta"`
	Message   *Message `json:"message"`
}

type Message struct {
	Role  string `json:"role"`
	Model string `json:"model"`
	Usage *Usage `json:"usage"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type ModelStats struct {
	InputTokens              int64
	OutputTokens             int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
	Messages                 int
}

func (s *ModelStats) add(u *Usage) {
	s.InputTokens += int64(u.InputTokens)
	s.OutputTokens += int64(u.OutputTokens)
	s.CacheCreationInputTokens += int64(u.CacheCreationInputTokens)
	s.CacheReadInputTokens += int64(u.CacheReadInputTokens)
	s.Messages++
}

func (s *ModelStats) merge(other *ModelStats) {
	s.InputTokens += other.InputTokens
	s.OutputTokens += other.OutputTokens
	s.CacheCreationInputTokens += other.CacheCreationInputTokens
	s.CacheReadInputTokens += other.CacheReadInputTokens
	s.Messages += other.Messages
}

func (s *ModelStats) totalTokens() int64 {
	return s.InputTokens + s.OutputTokens + s.CacheCreationInputTokens + s.CacheReadInputTokens
}

// Collected result
type Report struct {
	From    time.Time
	To      time.Time
	Days    int
	Models  map[string]*ModelStats            // model -> stats
	Daily   map[string]map[string]*ModelStats  // date -> model -> stats
	Grand   *ModelStats
	Sessions int
	Messages int
}

func main() {
	var (
		month     bool
		afterStr  string
		beforeStr string
		daily     bool
		modelF    string
		project   string
		jsonOut   bool
		noColor   bool
	)

	flag.BoolVar(&month, "month", false, "Last 30 days (default: last 7 days)")
	flag.StringVar(&afterStr, "after", "", "From date (YYYY-MM-DD)")
	flag.StringVar(&beforeStr, "before", "", "To date (YYYY-MM-DD)")
	flag.BoolVar(&daily, "daily", false, "Show daily breakdown")
	flag.StringVar(&modelF, "model", "", "Filter by model (partial match)")
	flag.StringVar(&project, "project", "workspace", `Project dir name (default: workspace, "all" for all)`)
	flag.BoolVar(&jsonOut, "json", false, "JSON output for AI cost calculation")
	flag.BoolVar(&noColor, "no-color", false, "Disable color output")
	flag.Parse()

	if noColor || jsonOut {
		disableColors()
	}

	// --- Date range ---
	loc := time.Now().Location()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var filterAfter, filterBefore time.Time
	var displayFrom, displayTo time.Time

	switch {
	case afterStr != "" || beforeStr != "":
		if afterStr != "" {
			t, err := time.ParseInLocation("2006-01-02", afterStr, loc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid -after date: %v\n", err)
				os.Exit(1)
			}
			filterAfter = t
			displayFrom = t
		}
		if beforeStr != "" {
			t, err := time.ParseInLocation("2006-01-02", beforeStr, loc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid -before date: %v\n", err)
				os.Exit(1)
			}
			filterBefore = t.Add(24*time.Hour - time.Nanosecond)
			displayTo = t
		}
		if displayFrom.IsZero() {
			displayFrom = displayTo
		}
		if displayTo.IsZero() {
			displayTo = today
			filterBefore = now.Add(time.Minute)
		}
	case month:
		displayFrom = today.AddDate(0, 0, -29)
		displayTo = today
		filterAfter = displayFrom
		filterBefore = now.Add(time.Minute)
	default:
		displayFrom = today.AddDate(0, 0, -6)
		displayTo = today
		filterAfter = displayFrom
		filterBefore = now.Add(time.Minute)
	}

	days := int(displayTo.Sub(displayFrom).Hours()/24) + 1
	if days < 1 {
		days = 1
	}

	// --- Find project dirs ---
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

	// --- Scan ---
	report := &Report{
		From:   displayFrom,
		To:     displayTo,
		Days:   days,
		Models: make(map[string]*ModelStats),
		Daily:  make(map[string]map[string]*ModelStats),
		Grand:  &ModelStats{},
	}
	sessions := make(map[string]bool)

	for _, dir := range projectDirs {
		// Main session files
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		for _, f := range files {
			scanFile(f, filterAfter, filterBefore, modelF, report, sessions)
		}
		// Subagent files
		subFiles, _ := filepath.Glob(filepath.Join(dir, "*/subagents/*.jsonl"))
		for _, f := range subFiles {
			scanFile(f, filterAfter, filterBefore, modelF, report, sessions)
		}
	}

	report.Sessions = len(sessions)

	// --- Output ---
	if jsonOut {
		outputJSON(report, daily)
	} else {
		outputHuman(report, daily)
	}
}

func scanFile(path string, after, before time.Time, modelFilter string, report *Report, sessions map[string]bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	fallbackSession := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Message == nil || entry.IsMeta {
			continue
		}
		if entry.Message.Role != "assistant" || entry.Message.Usage == nil {
			continue
		}

		// Parse timestamp
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

		model := normalizeModel(entry.Message.Model)
		if model == "" {
			model = "unknown"
		}

		// Model filter
		if modelFilter != "" && !strings.Contains(strings.ToLower(model), strings.ToLower(modelFilter)) {
			continue
		}

		usage := entry.Message.Usage

		// Skip zero-usage entries (e.g., <synthetic>)
		if usage.InputTokens == 0 && usage.OutputTokens == 0 &&
			usage.CacheCreationInputTokens == 0 && usage.CacheReadInputTokens == 0 {
			continue
		}

		// Session tracking
		sid := entry.SessionID
		if sid == "" {
			sid = fallbackSession
		}
		sessions[sid] = true
		report.Messages++

		// Model totals
		if report.Models[model] == nil {
			report.Models[model] = &ModelStats{}
		}
		report.Models[model].add(usage)
		report.Grand.add(usage)

		// Daily stats
		dateKey := ts.Local().Format("2006-01-02")
		if report.Daily[dateKey] == nil {
			report.Daily[dateKey] = make(map[string]*ModelStats)
		}
		if report.Daily[dateKey][model] == nil {
			report.Daily[dateKey][model] = &ModelStats{}
		}
		report.Daily[dateKey][model].add(usage)
	}
}

// normalizeModel strips the date suffix (e.g., "-20251101") from model IDs.
func normalizeModel(model string) string {
	if len(model) < 10 {
		return model
	}
	suffix := model[len(model)-9:]
	if suffix[0] != '-' {
		return model
	}
	for _, c := range suffix[1:] {
		if c < '0' || c > '9' {
			return model
		}
	}
	return model[:len(model)-9]
}

// --- Human output ---

func outputHuman(r *Report, showDaily bool) {
	fromStr := r.From.Format("2006-01-02")
	toStr := r.To.Format("2006-01-02")

	fmt.Printf("\n%s═══ Token Usage: %s ~ %s (%d days) ═══%s\n\n",
		colorBold, fromStr, toStr, r.Days, colorReset)

	if len(r.Models) == 0 {
		fmt.Printf("  %sNo usage data found for this period.%s\n\n", colorDim, colorReset)
		return
	}

	// Sort models by total tokens descending
	models := sortedModels(r.Models)

	// Table
	printTableHeader()
	for _, m := range models {
		printModelRow(m.name, m.stats, false)
	}
	if len(models) > 1 {
		printTableSep()
		printModelRow("Total", r.Grand, true)
	}

	fmt.Printf("\n  %sSessions: %d  |  Messages: %d  |  Period: %d days%s\n",
		colorDim, r.Sessions, r.Messages, r.Days, colorReset)

	// Daily breakdown
	if showDaily && len(r.Daily) > 0 {
		fmt.Printf("\n%s── Daily Breakdown ──%s\n\n", colorCyan, colorReset)

		dates := sortedDates(r.Daily)
		for _, date := range dates {
			dayModels := r.Daily[date]
			dayTotal := &ModelStats{}
			for _, s := range dayModels {
				dayTotal.merge(s)
			}

			fmt.Printf("  %s%s%s  in:%-11s  out:%-11s  cw:%-11s  cr:%-11s  %s(%d msgs)%s\n",
				colorBold, date, colorReset,
				fmtNum(dayTotal.InputTokens),
				fmtNum(dayTotal.OutputTokens),
				fmtNum(dayTotal.CacheCreationInputTokens),
				fmtNum(dayTotal.CacheReadInputTokens),
				colorDim, dayTotal.Messages, colorReset)

			// Per-model detail if multiple models that day
			if len(dayModels) > 1 {
				dms := sortedModels(dayModels)
				for _, dm := range dms {
					fmt.Printf("    %s%-20s%s  in:%-11s  out:%-11s\n",
						colorDim, dm.name, colorReset,
						fmtNum(dm.stats.InputTokens),
						fmtNum(dm.stats.OutputTokens))
				}
			}
		}
	}

	fmt.Printf("\n  %sCost estimate: run with -json and ask AI to calculate pricing%s\n\n",
		colorDim, colorReset)
}

func printTableHeader() {
	fmt.Printf("  %s%-22s  %12s  %12s  %12s  %12s%s\n",
		colorDim, "Model", "Input", "Output", "Cache Write", "Cache Read", colorReset)
	printTableSep()
}

func printTableSep() {
	fmt.Printf("  %s%s  %s  %s  %s  %s%s\n",
		colorDim,
		strings.Repeat("─", 22),
		strings.Repeat("─", 12),
		strings.Repeat("─", 12),
		strings.Repeat("─", 12),
		strings.Repeat("─", 12),
		colorReset)
}

func printModelRow(name string, s *ModelStats, bold bool) {
	nameColor := colorCyan
	if bold {
		nameColor = colorBold
	}
	fmt.Printf("  %s%-22s%s  %s%12s%s  %s%12s%s  %s%12s%s  %s%12s%s\n",
		nameColor, name, colorReset,
		colorGreen, fmtNum(s.InputTokens), colorReset,
		colorYellow, fmtNum(s.OutputTokens), colorReset,
		colorDim, fmtNum(s.CacheCreationInputTokens), colorReset,
		colorDim, fmtNum(s.CacheReadInputTokens), colorReset)
}

// --- JSON output ---

type JSONModelStats struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	Messages                 int   `json:"messages"`
}

type JSONDailyEntry struct {
	Date   string                    `json:"date"`
	Models map[string]*JSONModelStats `json:"models"`
	Total  *JSONModelStats           `json:"total"`
}

type JSONOutput struct {
	Period struct {
		From string `json:"from"`
		To   string `json:"to"`
		Days int    `json:"days"`
	} `json:"period"`
	Models   map[string]*JSONModelStats `json:"models"`
	Totals   *JSONModelStats            `json:"totals"`
	Sessions int                        `json:"sessions"`
	Messages int                        `json:"messages"`
	Daily    []JSONDailyEntry           `json:"daily,omitempty"`
}

func toJSONStats(s *ModelStats) *JSONModelStats {
	return &JSONModelStats{
		InputTokens:              s.InputTokens,
		OutputTokens:             s.OutputTokens,
		CacheCreationInputTokens: s.CacheCreationInputTokens,
		CacheReadInputTokens:     s.CacheReadInputTokens,
		Messages:                 s.Messages,
	}
}

func outputJSON(r *Report, showDaily bool) {
	out := JSONOutput{
		Models:   make(map[string]*JSONModelStats),
		Sessions: r.Sessions,
		Messages: r.Messages,
	}
	out.Period.From = r.From.Format("2006-01-02")
	out.Period.To = r.To.Format("2006-01-02")
	out.Period.Days = r.Days

	for name, s := range r.Models {
		out.Models[name] = toJSONStats(s)
	}
	out.Totals = toJSONStats(r.Grand)

	if showDaily {
		dates := sortedDates(r.Daily)
		for _, date := range dates {
			entry := JSONDailyEntry{
				Date:   date,
				Models: make(map[string]*JSONModelStats),
				Total:  &JSONModelStats{},
			}
			for model, s := range r.Daily[date] {
				entry.Models[model] = toJSONStats(s)
				entry.Total.InputTokens += s.InputTokens
				entry.Total.OutputTokens += s.OutputTokens
				entry.Total.CacheCreationInputTokens += s.CacheCreationInputTokens
				entry.Total.CacheReadInputTokens += s.CacheReadInputTokens
				entry.Total.Messages += s.Messages
			}
			out.Daily = append(out.Daily, entry)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

// --- Helpers ---

type modelEntry struct {
	name  string
	stats *ModelStats
}

func sortedModels(m map[string]*ModelStats) []modelEntry {
	entries := make([]modelEntry, 0, len(m))
	for name, stats := range m {
		entries = append(entries, modelEntry{name, stats})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].stats.totalTokens() > entries[j].stats.totalTokens()
	})
	return entries
}

func sortedDates(m map[string]map[string]*ModelStats) []string {
	dates := make([]string, 0, len(m))
	for d := range m {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates
}

func fmtNum(n int64) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+(len(s)-1)/3)
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
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

func disableColors() {
	colorReset = ""
	colorDim = ""
	colorBold = ""
	colorGreen = ""
	colorYellow = ""
	colorCyan = ""
}
