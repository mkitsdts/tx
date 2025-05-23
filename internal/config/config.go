package config

import (
	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Jaeger   JaegerConfig   `mapstructure:"jaeger"`
	Pprof    PprofConfig    `mapstructure:"pprof"`
}

// GRPCConfig gRPC服务器配置
type GRPCConfig struct {
	Address string `mapstructure:"address"`
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JaegerConfig Jaeger配置
type JaegerConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"service_name"`
}

// PprofConfig pprof配置
type PprofConfig struct {
	Address string `mapstructure:"address"`
}

// NewConfig 创建配置
func NewConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("grpc.address", ":50051")
	viper.SetDefault("pprof.address", ":6060")
	viper.SetDefault("postgres.sslmode", "disable")
	viper.SetDefault("jaeger.service_name", "tx-service")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
