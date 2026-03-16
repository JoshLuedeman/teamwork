package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [workflow-id]",
	Short: "View workflow activity logs",
	Long:  "Read and filter .teamwork/metrics/ JSONL log entries with human-readable formatting.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().StringP("role", "r", "", "Filter by role (e.g., coder, tester)")
	logsCmd.Flags().StringP("action", "a", "", "Filter by action (e.g., start, complete, gate_result)")
	logsCmd.Flags().IntP("tail", "n", 0, "Show only the last N entries")
	logsCmd.Flags().String("since", "", "Show entries since timestamp (ISO 8601 or duration like '24h', '7d')")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	roleFilter, _ := cmd.Flags().GetString("role")
	actionFilter, _ := cmd.Flags().GetString("action")
	tailN, _ := cmd.Flags().GetInt("tail")
	sinceStr, _ := cmd.Flags().GetString("since")

	var sinceTime time.Time
	if sinceStr != "" {
		sinceTime, err = parseSince(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: %w", sinceStr, err)
		}
	}

	metricsDir := filepath.Join(dir, ".teamwork", "metrics")

	var events []metrics.Event
	if len(args) == 1 {
		events, err = metrics.Load(dir, args[0])
		if err != nil {
			return fmt.Errorf("loading metrics: %w", err)
		}
	} else {
		events, err = loadAllEvents(metricsDir)
		if err != nil {
			return err
		}
	}

	if events == nil && !dirExists(metricsDir) {
		fmt.Fprintln(cmd.OutOrStdout(), "No metrics directory found. Run a workflow to generate activity logs.")
		return nil
	}

	filtered := filterEvents(events, roleFilter, actionFilter, sinceTime)

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Timestamp < filtered[j].Timestamp
	})

	if tailN > 0 && len(filtered) > tailN {
		filtered = filtered[len(filtered)-tailN:]
	}

	w := cmd.OutOrStdout()
	for _, ev := range filtered {
		ts := formatTimestamp(ev.Timestamp)
		wf := truncateLog(ev.Workflow, 24)
		fmt.Fprintf(w, "%s  %-24s  %-12s  %-10s  %s\n", ts, wf, ev.Action, ev.Role, ev.Detail)
	}

	return nil
}

func loadAllEvents(metricsDir string) ([]metrics.Event, error) {
	entries, err := os.ReadDir(metricsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading metrics directory: %w", err)
	}

	var all []metrics.Event
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		evs, err := loadEventsFromFile(filepath.Join(metricsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		all = append(all, evs...)
	}
	return all, nil
}

func loadEventsFromFile(path string) ([]metrics.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open metrics file %s: %w", path, err)
	}
	defer f.Close()

	var events []metrics.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev metrics.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, fmt.Errorf("unmarshal event in %s: %w", path, err)
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read metrics file %s: %w", path, err)
	}
	return events, nil
}

func filterEvents(events []metrics.Event, role, action string, since time.Time) []metrics.Event {
	if role == "" && action == "" && since.IsZero() {
		return events
	}

	var result []metrics.Event
	for _, ev := range events {
		if role != "" && ev.Role != role {
			continue
		}
		if action != "" && ev.Action != action {
			continue
		}
		if !since.IsZero() {
			evTime, err := time.Parse(time.RFC3339, ev.Timestamp)
			if err != nil {
				continue
			}
			if evTime.Before(since) {
				continue
			}
		}
		result = append(result, ev)
	}
	return result
}

var durationRegexp = regexp.MustCompile(`^(\d+)([hd])$`)

func parseSince(s string) (time.Time, error) {
	if m := durationRegexp.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "h":
			return time.Now().UTC().Add(-time.Duration(n) * time.Hour), nil
		case "d":
			return time.Now().UTC().Add(-time.Duration(n) * 24 * time.Hour), nil
		}
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected ISO 8601 date (e.g., 2026-03-01) or relative duration (e.g., 24h, 7d)")
}

func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}

func truncateLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
