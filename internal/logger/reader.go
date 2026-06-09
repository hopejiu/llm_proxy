package logger

import (
	"os"
	"strings"
	"sync"
)

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// LogReader 从日志文件增量读取日志条目
type LogReader struct {
	filePath string
	offset   int64
	mu       sync.Mutex
}

// NewLogReader 创建 LogReader，offset 初始化为 0
func NewLogReader(filePath string) *LogReader {
	return &LogReader{
		filePath: filePath,
		offset:   0,
	}
}

// ReadAllLogs 从文件头读取全部日志，更新 offset 到文件末尾
func (r *LogReader) ReadAllLogs() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil
	}

	r.offset = int64(len(data))

	return parseSlogLines(data)
}

// ReadNewLogs 从 offset 位置读取新增日志，更新 offset
func (r *LogReader) ReadNewLogs() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, err := os.Stat(r.filePath)
	if err != nil {
		return nil
	}

	fileSize := info.Size()
	if fileSize <= r.offset {
		return nil
	}

	f, err := os.Open(r.filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	buf := make([]byte, fileSize-r.offset)
	n, err := f.ReadAt(buf, r.offset)
	if err != nil && n == 0 {
		return nil
	}
	buf = buf[:n]

	r.offset += int64(n)

	return parseSlogLines(buf)
}

// ResetOffset 重置 offset 为 0（用于文件归档后重新读取）
func (r *LogReader) ResetOffset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.offset = 0
}

// parseSlogLines 解析 slog TextHandler 输出格式的多行文本
func parseSlogLines(data []byte) []LogEntry {
	text := string(data)
	lines := strings.Split(text, "\n")

	var entries []LogEntry
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		entry := parseSlogLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	return entries
}

// parseSlogLine 解析单行 slog TextHandler 输出
// 格式: time=2026-05-13T10:50:07.481+08:00 level=INFO msg="配置加载成功" db_type=mysql proxy_port=8888
func parseSlogLine(line string) *LogEntry {
	entry := &LogEntry{}

	levelStr := extractAttr(line, "level")
	if levelStr == "" {
		levelStr = "info"
	}
	entry.Level = strings.ToLower(levelStr)

	msgStr := extractAttr(line, "msg")
	if msgStr == "" {
		return nil
	}
	msgStr = strings.Trim(msgStr, "\"")
	entry.Message = msgStr

	timeStr := extractAttr(line, "time")
	if timeStr != "" {
		timeStr = strings.Trim(timeStr, "\"")
		entry.Time = formatTimeShort(timeStr)
	}

	extraAttrs := extractExtraAttrs(line, []string{"time", "level", "msg"})
	if extraAttrs != "" {
		entry.Message += " " + extraAttrs
	}

	return entry
}

// formatTimeShort 将 ISO 时间格式转为 HH:MM:SS.mmm
func formatTimeShort(timeStr string) string {
	if len(timeStr) < 19 {
		return timeStr
	}
	parts := strings.SplitN(timeStr, "T", 2)
	if len(parts) != 2 {
		return timeStr
	}
	timePart := parts[1]
	if idx := strings.Index(timePart, "+"); idx > 0 {
		timePart = timePart[:idx]
	}
	if idx := strings.Index(timePart, "-"); idx > 4 {
		timePart = timePart[:idx]
	}
	if idx := strings.Index(timePart, "."); idx > 0 && idx+4 <= len(timePart) {
		timePart = timePart[:idx+4]
	}
	return timePart
}

// extractAttr 从 slog TextHandler 格式的行中提取指定属性值
func extractAttr(line, key string) string {
	prefix := key + "="
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}

	start := idx + len(prefix)
	if start >= len(line) {
		return ""
	}

	if line[start] == '"' {
		end := start + 1
		for end < len(line) {
			if line[end] == '"' && (end+1 >= len(line) || line[end+1] == ' ') {
				return line[start : end+1]
			}
			end++
		}
		return line[start:]
	}

	end := start
	for end < len(line) && line[end] != ' ' {
		end++
	}
	return line[start:end]
}

// extractExtraAttrs 提取除 skipKeys 外的所有属性
func extractExtraAttrs(line string, skipKeys []string) string {
	var result strings.Builder
	remaining := line

	for {
		eqIdx := strings.Index(remaining, "=")
		if eqIdx < 0 {
			break
		}

		keyStart := eqIdx - 1
		for keyStart >= 0 && remaining[keyStart] != ' ' {
			keyStart--
		}
		keyStart++

		key := remaining[keyStart:eqIdx]

		skip := false
		for _, sk := range skipKeys {
			if key == sk {
				skip = true
				break
			}
		}

		valStart := eqIdx + 1
		if valStart >= len(remaining) {
			break
		}

		var valEnd int
		if remaining[valStart] == '"' {
			valEnd = valStart + 1
			for valEnd < len(remaining) {
				if remaining[valEnd] == '"' && (valEnd+1 >= len(remaining) || remaining[valEnd+1] == ' ') {
					valEnd++
					break
				}
				valEnd++
			}
		} else {
			valEnd = valStart
			for valEnd < len(remaining) && remaining[valEnd] != ' ' {
				valEnd++
			}
		}

		if !skip {
			if result.Len() > 0 {
				result.WriteByte(' ')
			}
			result.WriteString(key)
			result.WriteByte('=')
			result.WriteString(remaining[valStart:valEnd])
		}

		if valEnd < len(remaining) && remaining[valEnd] == ' ' {
			remaining = remaining[valEnd+1:]
		} else {
			break
		}
	}

	return result.String()
}