// internal/config/config.go
package config

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPPort     string        `mapstructure:"http_port"`
	DBUrl        string        `mapstructure:"db_url"`
	RedisAddr    string        `mapstructure:"redis_addr"`
	KafkaBrokers []string      `mapstructure:"kafka_brokers"`
	JWTSecret    string        `mapstructure:"jwt_secret"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

func Load() *Config {
	viper.SetDefault("http_port", ":8080")
	viper.SetDefault("db_url", "postgres://postgres:postgres@localhost:5432/furniture?sslmode=disable")
	viper.SetDefault("redis_addr", "localhost:6379")
	viper.SetDefault("kafka_brokers", []string{"localhost:9092"})
	viper.SetDefault("jwt_secret", "talesofrussianglubinka")
	viper.SetDefault("read_timeout", 15*time.Second)
	viper.SetDefault("write_timeout", 15*time.Second)
	viper.SetDefault("idle_timeout", 60*time.Second)

	viper.SetConfigName("config")                    // name of config file (without extension)
	viper.SetConfigType("yaml")                      // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/var/furniture-shop-api/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.furniture-shop-api") // call multiple times to add many search paths
	viper.AddConfigPath(".")                         // optionally look for config in the working directory

	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("Config file not found, using defaults and environment variables")
		} else {
			log.Printf("Error reading config file: %v", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("fatal error unmarshaling config: %w", err))
	}

	return &cfg
}
