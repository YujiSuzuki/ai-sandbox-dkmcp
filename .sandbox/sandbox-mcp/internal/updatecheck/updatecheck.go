package updatecheck

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// TemplateConfig holds the template repository configuration
type TemplateConfig struct {
	Repo          string
	Channel       string
	Enabled       bool
	IntervalHours int
}

// UpdateStatus holds the current update status
type UpdateStatus struct {
	LatestVersion string
	Repo          string
	Channel       string
	ReleaseURL    string
	Enabled       bool
}

// ReadStateFile reads the update check state file
// Format: <unix_timestamp>:<version>
// Returns the latest version string, or empty string if file doesn't exist (first run)
func ReadStateFile(stateFile string) (string, error) {
	// Check if file exists
	info, err := os.Stat(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// First run, no state file yet
			return "", nil
		}
		return "", fmt.Errorf("failed to stat state file: %w", err)
	}

	// Check if file is empty
	if info.Size() == 0 {
		return "", fmt.Errorf("state file is empty")
	}

	// Read file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read state file: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("state file is empty")
	}

	// Parse format: <timestamp>:<version>
	parts := strings.SplitN(content, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid state file format (expected timestamp:version)")
	}

	version := parts[1]
	return version, nil
}

// ParseTemplateConfig parses the template configuration file
func ParseTemplateConfig(configFile string) (*TemplateConfig, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := &TemplateConfig{
		Channel:       "all",   // default
		Enabled:       true,    // default
		IntervalHours: 24,      // default
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY="VALUE" or KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes
		value = strings.Trim(value, `"'`)

		switch key {
		case "TEMPLATE_REPO":
			cfg.Repo = value
		case "CHECK_CHANNEL":
			cfg.Channel = value
		case "CHECK_UPDATES":
			cfg.Enabled = (value == "true")
		case "CHECK_INTERVAL_HOURS":
			hours, err := strconv.Atoi(value)
			if err == nil {
				cfg.IntervalHours = hours
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Validate required fields
	if cfg.Repo == "" {
		return nil, fmt.Errorf("TEMPLATE_REPO is required in config file")
	}

	return cfg, nil
}

// GetUpdateStatus returns the current update status
func GetUpdateStatus(stateFile, configFile string) (*UpdateStatus, error) {
	// Read config
	cfg, err := ParseTemplateConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Read state
	version, err := ReadStateFile(stateFile)
	if err != nil {
		// If state file is corrupted, return config info only
		version = ""
	}

	// Build status
	status := &UpdateStatus{
		LatestVersion: version,
		Repo:          cfg.Repo,
		Channel:       cfg.Channel,
		Enabled:       cfg.Enabled,
		ReleaseURL:    fmt.Sprintf("https://github.com/%s/releases", cfg.Repo),
	}

	return status, nil
}
