// config 包负责从 config.yaml 读取配置，并提供全局配置访问入口。
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port int `mapstructure:"port"` // 监听端口
}

// DatabaseConfig 数据库连接配置。
type DatabaseConfig struct {
	Host      string `mapstructure:"host"`      // 数据库主机地址
	Port      int    `mapstructure:"port"`      // 数据库端口
	User      string `mapstructure:"user"`      // 数据库用户名
	Password  string `mapstructure:"password"`  // 数据库密码
	DBName    string `mapstructure:"dbname"`    // 数据库名称
	Charset   string `mapstructure:"charset"`   // 字符集
	ParseTime bool   `mapstructure:"parseTime"` // 是否解析时间字段
	Loc       string `mapstructure:"loc"`       // 时区
}

// JWTConfig JWT 签发与校验配置。
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`       // 签名密钥
	ExpireHours int    `mapstructure:"expire_hours"` // Token 有效小时数
}

// Config 全局配置聚合结构体。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

// AppConfig 全局配置实例，在 LoadConfig 或 SetTestConfig 后可用。
var AppConfig *Config

// SetTestConfig 供测试使用，直接注入配置，无需读取 config.yaml 文件。
func SetTestConfig(cfg *Config) {
	AppConfig = cfg
}

// LoadConfig 使用 viper 从当前目录的 config.yaml 读取并解析配置。
func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// DSN 生成 MySQL 连接字符串。
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset, d.ParseTime, d.Loc,
	)
}
