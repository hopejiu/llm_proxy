package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBType     string // mysql 或 sqlite
	DBPath     string // SQLite数据库文件路径
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ProxyPort  string

	// 超时与重试
	HTTPTimeout            time.Duration // HTTP 请求超时
	StreamFirstByteTimeout time.Duration // 流式首次数据超时
	StreamMaxRetries       int           // 流式最大重试次数
	RetryDelayBase         time.Duration // 重试延迟基数

	// 缓存与清理
	ProviderCacheTTL time.Duration // Provider 缓存 TTL
	LogCleanupDays   int           // 日志清理天数

	// 日志
	LogLevel string // 日志级别: debug, info, warn, error

	// 桌面应用配置
	AutoStartProxy bool // 启动时是否自动启动代理服务
}

// DataDir 返回应用数据目录（%APPDATA%/llm-proxy/），并确保目录存在
func DataDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		// 非 Windows 或 APPDATA 未设置，回退到用户主目录
		home, _ := os.UserHomeDir()
		appData = home
	}
	dir := filepath.Join(appData, "llm-proxy")
	os.MkdirAll(dir, 0755)
	return dir
}

func Load() *Config {
	// 从应用数据目录加载 .env
	envPath := filepath.Join(DataDir(), ".env")
	godotenv.Load(envPath)
	// 也尝试从当前工作目录加载（开发模式兼容）
	godotenv.Load()

	cfg := &Config{
		DBType:     getEnv("DB_TYPE", "mysql"),
		DBPath:     getEnv("DB_PATH", "llm_proxy.db"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "llm_proxy"),
		ProxyPort:  getEnv("PROXY_PORT", "8888"),

		HTTPTimeout:            getEnvDuration("HTTP_TIMEOUT", 300*time.Second),
		StreamFirstByteTimeout: getEnvDuration("STREAM_FIRST_BYTE_TIMEOUT", 5*time.Second),
		StreamMaxRetries:       getEnvInt("STREAM_MAX_RETRIES", 10),
		RetryDelayBase:         getEnvDuration("RETRY_DELAY_BASE", 500*time.Millisecond),
		ProviderCacheTTL:       getEnvDuration("PROVIDER_CACHE_TTL", 30*time.Second),
		LogCleanupDays:         getEnvInt("LOG_CLEANUP_DAYS", 14),
		LogLevel:               getEnv("LOG_LEVEL", "info"),

		AutoStartProxy: getEnvBool("AUTO_START_PROXY", true),
	}

	// SQLite 路径：如果是相对路径，改为基于应用数据目录
	if cfg.IsSQLite() && !filepath.IsAbs(cfg.DBPath) {
		cfg.DBPath = filepath.Join(DataDir(), cfg.DBPath)
	}

	cfg.validate()

	return cfg
}

// validate 校验配置合法性
func (c *Config) validate() {
	if !c.IsSQLite() {
		if c.DBUser == "" {
			slog.Warn("MySQL配置缺少DB_USER，使用默认值root")
		}
		if c.DBName == "" {
			slog.Warn("MySQL配置缺少DB_NAME，使用默认值llm_proxy")
		}
	}

	if err := validatePort(c.ProxyPort, "PROXY_PORT"); err != nil {
		slog.Warn("端口配置无效", "key", "PROXY_PORT", "value", c.ProxyPort, "error", err)
	}
}

func validatePort(port string, name string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("%s 不是有效的端口号", port)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("%s 超出有效端口范围(1-65535)", port)
	}
	return nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true&timeout=5s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// IsSQLite 判断是否使用SQLite数据库
func (c *Config) IsSQLite() bool {
	return c.DBType == "sqlite"
}

// SQLitePath 返回SQLite数据库文件路径（用于 os.Stat 等文件操作）
func (c *Config) SQLitePath() string {
	return c.DBPath
}

// SQLiteDSN 返回SQLite的DSN（含PRAGMA优化参数）
func (c *Config) SQLiteDSN() string {
	return c.DBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-64000)"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
		slog.Warn("配置项不是有效整数，使用默认值", "key", key, "value", value, "default", defaultValue)
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
		slog.Warn("配置项不是有效时长，使用默认值", "key", key, "value", value, "default", defaultValue)
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
		slog.Warn("配置项不是有效布尔值，使用默认值", "key", key, "value", value, "default", defaultValue)
	}
	return defaultValue
}

// EnvFilePath 返回 .env 文件路径
func EnvFilePath() string {
	return filepath.Join(DataDir(), ".env")
}

// EnvFileExists 检查 .env 文件是否存在
func EnvFileExists() bool {
	_, err := os.Stat(EnvFilePath())
	return err == nil
}

// EnvSelectOption 下拉选项
type EnvSelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// EnvItem 环境变量配置项
type EnvItem struct {
	Key          string            `json:"key"`
	Label        string            `json:"label"`
	Value        string            `json:"value"`
	DefaultValue string            `json:"default_value"`
	Type         string            `json:"type"` // text, number, bool, duration, select, password
	Group        string            `json:"group"`
	Description  string            `json:"description"`
	Options      []EnvSelectOption `json:"options,omitempty"`      // select 类型的选项列表
	DependsOn    string            `json:"depends_on,omitempty"`   // 条件显示：当指定 key 的值等于 DependsOnValue 时显示
	DependsValue string            `json:"depends_value,omitempty"` // DependsOn 匹配的值
}

// GetEnvItems 获取所有可配置的环境变量项（当前值 + 默认值 + 元信息）
func GetEnvItems() []EnvItem {
	items := []EnvItem{
		{Key: "DB_TYPE", Label: "数据库类型", Value: getEnv("DB_TYPE", "mysql"), DefaultValue: "mysql", Type: "select", Group: "数据库", Description: "选择数据库类型",
			Options: []EnvSelectOption{{Value: "mysql", Label: "MySQL"}, {Value: "sqlite", Label: "SQLite"}}},
		{Key: "DB_PATH", Label: "SQLite路径", Value: getEnv("DB_PATH", "llm_proxy.db"), DefaultValue: "llm_proxy.db", Type: "text", Group: "数据库", Description: "SQLite 数据库文件路径",
			DependsOn: "DB_TYPE", DependsValue: "sqlite"},
		{Key: "DB_HOST", Label: "MySQL主机", Value: getEnv("DB_HOST", "localhost"), DefaultValue: "localhost", Type: "text", Group: "数据库", Description: "MySQL 主机地址",
			DependsOn: "DB_TYPE", DependsValue: "mysql"},
		{Key: "DB_PORT", Label: "MySQL端口", Value: getEnv("DB_PORT", "3306"), DefaultValue: "3306", Type: "number", Group: "数据库", Description: "MySQL 端口号",
			DependsOn: "DB_TYPE", DependsValue: "mysql"},
		{Key: "DB_USER", Label: "MySQL用户", Value: getEnv("DB_USER", "root"), DefaultValue: "root", Type: "text", Group: "数据库", Description: "MySQL 用户名",
			DependsOn: "DB_TYPE", DependsValue: "mysql"},
		{Key: "DB_PASSWORD", Label: "MySQL密码", Value: getEnv("DB_PASSWORD", ""), DefaultValue: "", Type: "password", Group: "数据库", Description: "MySQL 密码",
			DependsOn: "DB_TYPE", DependsValue: "mysql"},
		{Key: "DB_NAME", Label: "MySQL库名", Value: getEnv("DB_NAME", "llm_proxy"), DefaultValue: "llm_proxy", Type: "text", Group: "数据库", Description: "MySQL 数据库名",
			DependsOn: "DB_TYPE", DependsValue: "mysql"},
		{Key: "PROXY_PORT", Label: "代理端口", Value: getEnv("PROXY_PORT", "8888"), DefaultValue: "8888", Type: "number", Group: "代理服务", Description: "LLM 代理服务监听端口"},
		{Key: "HTTP_TIMEOUT", Label: "HTTP超时", Value: getEnv("HTTP_TIMEOUT", "5m0s"), DefaultValue: "5m0s", Type: "select", Group: "代理服务", Description: "HTTP 请求超时时间",
			Options: []EnvSelectOption{
				{Value: "30s", Label: "30秒"}, {Value: "1m0s", Label: "1分钟"}, {Value: "2m0s", Label: "2分钟"},
				{Value: "5m0s", Label: "5分钟"}, {Value: "10m0s", Label: "10分钟"},
			}},
		{Key: "STREAM_FIRST_BYTE_TIMEOUT", Label: "流式首字节超时", Value: getEnv("STREAM_FIRST_BYTE_TIMEOUT", "5s"), DefaultValue: "5s", Type: "select", Group: "代理服务", Description: "流式请求等待首个数据的超时时间",
			Options: []EnvSelectOption{
				{Value: "3s", Label: "3秒"}, {Value: "5s", Label: "5秒"}, {Value: "10s", Label: "10秒"}, {Value: "15s", Label: "15秒"}, {Value: "30s", Label: "30秒"},
			}},
		{Key: "STREAM_MAX_RETRIES", Label: "流式最大重试", Value: getEnv("STREAM_MAX_RETRIES", "10"), DefaultValue: "10", Type: "select", Group: "代理服务", Description: "流式请求最大重试次数",
			Options: []EnvSelectOption{
				{Value: "0", Label: "不重试"}, {Value: "3", Label: "3次"}, {Value: "5", Label: "5次"}, {Value: "10", Label: "10次"}, {Value: "20", Label: "20次"},
			}},
		{Key: "RETRY_DELAY_BASE", Label: "重试延迟基数", Value: getEnv("RETRY_DELAY_BASE", "500ms"), DefaultValue: "500ms", Type: "select", Group: "代理服务", Description: "重试延迟基数",
			Options: []EnvSelectOption{
				{Value: "100ms", Label: "100毫秒"}, {Value: "300ms", Label: "300毫秒"}, {Value: "500ms", Label: "500毫秒"}, {Value: "1s", Label: "1秒"}, {Value: "2s", Label: "2秒"},
			}},
		{Key: "PROVIDER_CACHE_TTL", Label: "Provider缓存TTL", Value: getEnv("PROVIDER_CACHE_TTL", "30s"), DefaultValue: "30s", Type: "select", Group: "代理服务", Description: "Provider 列表缓存过期时间",
			Options: []EnvSelectOption{
				{Value: "10s", Label: "10秒"}, {Value: "30s", Label: "30秒"}, {Value: "1m0s", Label: "1分钟"}, {Value: "5m0s", Label: "5分钟"},
			}},
		{Key: "LOG_CLEANUP_DAYS", Label: "日志清理天数", Value: getEnv("LOG_CLEANUP_DAYS", "14"), DefaultValue: "14", Type: "select", Group: "其他", Description: "自动清理多少天前的日志数据",
			Options: []EnvSelectOption{
				{Value: "7", Label: "7天"}, {Value: "14", Label: "14天"}, {Value: "30", Label: "30天"}, {Value: "60", Label: "60天"}, {Value: "90", Label: "90天"},
			}},
		{Key: "LOG_LEVEL", Label: "日志级别", Value: getEnv("LOG_LEVEL", "info"), DefaultValue: "info", Type: "select", Group: "其他", Description: "日志级别",
			Options: []EnvSelectOption{
				{Value: "debug", Label: "Debug"}, {Value: "info", Label: "Info"}, {Value: "warn", Label: "Warn"}, {Value: "error", Label: "Error"},
			}},
		{Key: "AUTO_START_PROXY", Label: "自动启动代理", Value: getEnv("AUTO_START_PROXY", "true"), DefaultValue: "true", Type: "bool", Group: "其他", Description: "启动程序时是否自动启动代理服务"},
	}
	return items
}

// SaveEnvItems 将配置项保存到 .env 文件
func SaveEnvItems(items map[string]string) error {
	envPath := EnvFilePath()

	// 读取现有 .env 文件内容
	existing := make(map[string]string)
	if data, err := os.ReadFile(envPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existing[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// 合并新值
	for k, v := range items {
		existing[k] = v
	}

	// 定义写入顺序（保持和 GetEnvItems 一致）
	keyOrder := []string{
		"DB_TYPE", "DB_PATH", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"PROXY_PORT", "HTTP_TIMEOUT", "STREAM_FIRST_BYTE_TIMEOUT", "STREAM_MAX_RETRIES",
		"RETRY_DELAY_BASE", "PROVIDER_CACHE_TTL", "LOG_CLEANUP_DAYS", "LOG_LEVEL", "AUTO_START_PROXY",
	}

	// 构建 .env 文件内容
	var lines []string
	lines = append(lines, "# LLM Proxy 配置文件")
	lines = append(lines, "# 修改后需重启程序生效（部分配置如日志级别可实时生效）")
	lines = append(lines, "")

	written := make(map[string]bool)
	for _, key := range keyOrder {
		if val, ok := existing[key]; ok {
			lines = append(lines, fmt.Sprintf("%s=%s", key, val))
			written[key] = true
		}
	}
	// 写入不在 keyOrder 中的其他键
	for key, val := range existing {
		if !written[key] {
			lines = append(lines, fmt.Sprintf("%s=%s", key, val))
		}
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(envPath, []byte(content), 0644)
}
