// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"annave.tech/cli/internal/log/domain"
)

var (
	// nginxRe matches standard nginx combined log format lines.
	nginxRe = regexp.MustCompile(`^(\S+) - (\S+) \[([^\]]+)\] "([^"]*)" (\d+) (\d+)`)
	// syslogRe matches BSD syslog lines with host and process fields.
	syslogRe = regexp.MustCompile(`^([A-Z][a-z]{2}\s+\d+\s+\d{2}:\d{2}:\d{2})\s+\S+\s+\S+:\s+(.+)$`)
	// isoTimeRe captures an ISO-8601 or RFC-3339 timestamp at the start of a line.
	isoTimeRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?)`)
)

var levelKeywords = []string{"CRITICAL", "FATAL", "ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE"}

// detectFormat identifies the log format from the first non-empty line.
func detectFormat(line string) domain.LogFormat {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "{") {
		var m map[string]interface{}
		if json.Unmarshal([]byte(trimmed), &m) == nil {
			return domain.FormatJSON
		}
	}
	if nginxRe.MatchString(trimmed) {
		return domain.FormatNginx
	}
	if syslogRe.MatchString(trimmed) {
		return domain.FormatSyslog
	}
	return domain.FormatPlain
}

// parseLine parses a single log line using the given format hint.
func parseLine(line string, lineNum int, format domain.LogFormat) domain.LogEntry {
	entry := domain.LogEntry{Line: lineNum, Raw: line, Format: format}
	switch format {
	case domain.FormatJSON:
		parseJSON(&entry, line)
	case domain.FormatNginx:
		parseNginx(&entry, line)
	case domain.FormatSyslog:
		parseSyslog(&entry, line)
	default:
		parsePlain(&entry, line)
	}
	return entry
}

func parseJSON(e *domain.LogEntry, line string) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		e.Message = line
		return
	}
	for _, key := range []string{"time", "timestamp", "ts", "@timestamp"} {
		if v, ok := m[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				if t, err := parseTimeStr(s); err == nil {
					e.Timestamp = t
					break
				}
			}
			var f float64
			if json.Unmarshal(v, &f) == nil {
				e.Timestamp = time.Unix(int64(f), 0)
				break
			}
		}
	}
	for _, key := range []string{"level", "severity", "lvl", "loglevel"} {
		if v, ok := m[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				e.Level = normalizeLevel(s)
				break
			}
		}
	}
	for _, key := range []string{"msg", "message", "text", "body"} {
		if v, ok := m[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				e.Message = s
				break
			}
		}
	}
	if e.Message == "" {
		e.Message = line
	}
}

func parseNginx(e *domain.LogEntry, line string) {
	m := nginxRe.FindStringSubmatch(line)
	if len(m) < 6 {
		e.Message = line
		return
	}
	if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[3]); err == nil {
		e.Timestamp = t
	}
	status := m[5]
	e.Message = m[4] + " " + status
	switch {
	case strings.HasPrefix(status, "5"):
		e.Level = "error"
	case strings.HasPrefix(status, "4"):
		e.Level = "warn"
	default:
		e.Level = "info"
	}
}

func parseSyslog(e *domain.LogEntry, line string) {
	m := syslogRe.FindStringSubmatch(line)
	if len(m) < 3 {
		e.Message = line
		return
	}
	ts := strings.TrimSpace(m[1])
	t, err := time.Parse("Jan  2 15:04:05", ts)
	if err != nil {
		t, err = time.Parse("Jan 02 15:04:05", ts)
	}
	if err == nil {
		now := time.Now()
		e.Timestamp = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
	}
	e.Message = m[2]
	e.Level = extractLevel(e.Message)
}

func parsePlain(e *domain.LogEntry, line string) {
	if m := isoTimeRe.FindString(line); m != "" {
		if t, err := parseTimeStr(m); err == nil {
			e.Timestamp = t
			line = strings.TrimSpace(line[len(m):])
		}
	}
	e.Level = extractLevel(line)
	e.Message = line
}

func extractLevel(s string) string {
	upper := strings.ToUpper(s)
	for _, kw := range levelKeywords {
		if strings.Contains(upper, kw) {
			return normalizeLevel(kw)
		}
	}
	return "info"
}

func normalizeLevel(s string) string {
	switch strings.ToUpper(s) {
	case "CRITICAL", "FATAL", "EMERG", "ALERT", "CRIT":
		return "critical"
	case "ERROR", "ERR":
		return "error"
	case "WARN", "WARNING":
		return "warn"
	case "INFO", "NOTICE":
		return "info"
	case "DEBUG", "TRACE", "VERBOSE":
		return "debug"
	}
	return strings.ToLower(s)
}

func parseTimeStr(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

// parseLines reads up to maxLines from r and returns all successfully parsed entries.
func parseLines(r io.Reader, maxLines int) (entries []domain.LogEntry, format domain.LogFormat, totalLines int, parseErrors int) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		totalLines++
		if maxLines > 0 && totalLines > maxLines {
			break
		}
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if format == "" {
			format = detectFormat(line)
		}
		e := parseLine(line, totalLines, format)
		if e.Message == "" {
			parseErrors++
			continue
		}
		entries = append(entries, e)
	}

	if format == "" {
		format = domain.FormatPlain
	}
	return
}
