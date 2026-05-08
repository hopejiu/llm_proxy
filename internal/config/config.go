package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
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
	WebPort    string
	ProxyPort  string

	// 超时与重试
	HTTPTimeout           time.Duration // HTTP 请求超时
	StreamFirstByteTimeout time.Duration // 流式首次数据超时
	StreamMaxRetries      int           // 流式最大重试次数
	RetryDelayBase        time.Duration // 重试延迟基数

	// 缓存与清理
	ProviderCacheTTL time.Duration // Provider 缓存 TTL
	LogCleanupDays   int           // 日志清理天数
}

func Load() *Config {
	// 尝试加载.env文件，如果不存在则忽略
	godotenv.Load()

	cfg := &Config{
		DBType:     getEnv("DB_TYPE", "mysql"),
		DBPath:     getEnv("DB_PATH", "llm_proxy.db"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "llm_proxy"),
		WebPort:    getEnv("WEB_PORT", "80"),
		ProxyPort:  getEnv("PROXY_PORT", "8888"),

		HTTPTimeout:            getEnvDuration("HTTP_TIMEOUT", 300*time.Second),
		StreamFirstByteTimeout: getEnvDuration("STREAM_FIRST_BYTE_TIMEOUT", 5*time.Second),
		StreamMaxRetries:       getEnvInt("STREAM_MAX_RETRIES", 10),
		RetryDelayBase:         getEnvDuration("RETRY_DELAY_BASE", 500*time.Millisecond),
		ProviderCacheTTL:       getEnvDuration("PROVIDER_CACHE_TTL", 30*time.Second),
		LogCleanupDays:         getEnvInt("LOG_CLEANUP_DAYS", 14),
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

	if err := validatePort(c.WebPort, "WEB_PORT"); err != nil {
		slog.Warn("端口配置无效", "key", "WEB_PORT", "value", c.WebPort, "error", err)
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
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// IsSQLite 判断是否使用SQLite数据库
func (c *Config) IsSQLite() bool {
	return c.DBType == "sqlite"
}

// SQLiteDSN 返回SQLite的DSN
func (c *Config) SQLiteDSN() string {
	return c.DBPath
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
