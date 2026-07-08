package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

type RunMetrics struct {
	UserID            string
	WindowHours       int
	RunsTotal         int
	RunsSucceeded     int
	RunsFailed        int
	RunsRunning       int
	RunSuccessRate    float64
	AvgRunLatencyMS   int64
	P95RunLatencyMS   int64
	ToolsTotal        int
	ToolsFailed       int
	ToolsBlocked      int
	ToolErrorRate     float64
	AvgToolDurationMS int64
}

func GetRunMetrics(ctx context.Context, db *sql.DB, userID string, windowHours int) (RunMetrics, error) {
	if db == nil {
		return RunMetrics{}, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	if windowHours <= 0 {
		windowHours = 24
	}
	since := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	out := RunMetrics{UserID: userID, WindowHours: windowHours}

	rows, err := db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM runs WHERE user_id=? AND started_at>=? GROUP BY status`,
		userID, since,
	)
	if err != nil {
		return out, fmt.Errorf("query run metrics: %w", err)
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return out, err
		}
		out.RunsTotal += count
		switch status {
		case RunStatusSucceeded:
			out.RunsSucceeded = count
		case RunStatusFailed:
			out.RunsFailed = count
		case RunStatusRunning:
			out.RunsRunning = count
		}
	}
	if err := rows.Close(); err != nil {
		return out, err
	}
	if out.RunsTotal > 0 {
		out.RunSuccessRate = float64(out.RunsSucceeded) / float64(out.RunsTotal)
	}

	latencies, err := runLatencies(ctx, db, userID, since)
	if err != nil {
		return out, err
	}
	out.AvgRunLatencyMS = averageInt64(latencies)
	out.P95RunLatencyMS = percentileInt64(latencies, 0.95)

	if err := fillToolMetrics(ctx, db, userID, since, &out); err != nil {
		return out, err
	}
	return out, nil
}

func runLatencies(ctx context.Context, db *sql.DB, userID string, since time.Time) ([]int64, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT TIMESTAMPDIFF(MICROSECOND, started_at, ended_at) / 1000
		 FROM runs
		 WHERE user_id=? AND started_at>=? AND ended_at IS NOT NULL`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("query run latencies: %w", err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var latency int64
		if err := rows.Scan(&latency); err != nil {
			return nil, err
		}
		if latency >= 0 {
			out = append(out, latency)
		}
	}
	return out, rows.Err()
}

func fillToolMetrics(ctx context.Context, db *sql.DB, userID string, since time.Time, out *RunMetrics) error {
	rows, err := db.QueryContext(ctx,
		`SELECT status, COUNT(*), COALESCE(AVG(duration_ms), 0)
		 FROM tool_audit_logs
		 WHERE user_id=? AND created_at>=?
		 GROUP BY status`,
		userID, since,
	)
	if err != nil {
		return fmt.Errorf("query tool metrics: %w", err)
	}
	defer rows.Close()

	var weightedDuration int64
	for rows.Next() {
		var status string
		var count int
		var avgDuration float64
		if err := rows.Scan(&status, &count, &avgDuration); err != nil {
			return err
		}
		out.ToolsTotal += count
		weightedDuration += int64(math.Round(avgDuration * float64(count)))
		switch status {
		case ToolStatusFailed:
			out.ToolsFailed = count
		case ToolStatusBlocked:
			out.ToolsBlocked = count
		}
	}
	if out.ToolsTotal > 0 {
		out.ToolErrorRate = float64(out.ToolsFailed+out.ToolsBlocked) / float64(out.ToolsTotal)
		out.AvgToolDurationMS = weightedDuration / int64(out.ToolsTotal)
	}
	return rows.Err()
}

func averageInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var total int64
	for _, value := range values {
		total += value
	}
	return total / int64(len(values))
}

func percentileInt64(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
