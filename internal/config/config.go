package config

import (
	"fmt"
	"os"

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
}

func Load() *Config {
	// 尝试加载.env文件，如果不存在则忽略
	godotenv.Load()

	return &Config{
		DBType:     getEnv("DB_TYPE", "mysql"),
		DBPath:     getEnv("DB_PATH", "llm_proxy.db"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "wang"),
		DBName:     getEnv("DB_NAME", "llm_proxy"),
		WebPort:    getEnv("WEB_PORT", "80"),
		ProxyPort:  getEnv("PROXY_PORT", "8888"),
	}
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
