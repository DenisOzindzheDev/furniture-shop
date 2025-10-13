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
	CorsDebug    bool          `mapstructure:"cors_debug"`

	MaxUploadSize     int64    `mapstructure:"max_upload_size"`
	AllowedImageTypes []string `mapstructure:"allowed_image_types"`

	AWS AWS `mapstructure:"aws"`
	PDF PDF `mapstructure:"pdf"`
}

type AWS struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	S3Bucket        string `mapstructure:"s3_bucket"`
	S3Host          string `mapstructure:"s3_host"`
}

type PDF struct {
	BaseURL     string `mapstructure:"base_url"`
	FontPath    string `mapstructure:"font_path"`
	LogoPath    string `mapstructure:"logo_path"`
	CompanyName string `mapstructure:"company_name"`
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
	viper.SetDefault("cors_debug", true)
	viper.SetDefault("max_upload_size", 10485760) // 10MB
	viper.SetDefault("allowed_image_types", []string{"image/jpeg", "image/png", "image/webp"})
	viper.SetDefault("aws.region", "us-east-1")
	viper.SetDefault("aws.access_key_id", "furniture")
	viper.SetDefault("aws.secret_access_key", "furniture")
	viper.SetDefault("aws.s3_bucket", "furniture")
	viper.SetDefault("aws.s3_host", "furniture-s3")
	viper.SetDefault("pdf.base_url", "http://localhost:8080")
	viper.SetDefault("pdf.company_name", "Furniture Shop")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/var/furniture-shop-api/")
	viper.AddConfigPath("$HOME/.furniture-shop-api")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	viper.BindEnv("http_port", "APP_HTTP_PORT")
	viper.BindEnv("db_url", "APP_DB_URL")
	viper.BindEnv("redis_addr", "APP_REDIS_ADDR")
	viper.BindEnv("kafka_brokers", "APP_KAFKA_BROKERS")
	viper.BindEnv("jwt_secret", "APP_JWT_SECRET")
	viper.BindEnv("read_timeout", "APP_READ_TIMEOUT")
	viper.BindEnv("write_timeout", "APP_WRITE_TIMEOUT")
	viper.BindEnv("idle_timeout", "APP_IDLE_TIMEOUT")
	viper.BindEnv("cors_debug", "APP_CORS_DEBUG")
	viper.BindEnv("max_upload_size", "APP_MAX_UPLOAD_SIZE")
	viper.BindEnv("allowed_image_types", "APP_ALLOWED_IMAGE_TYPES")
	viper.BindEnv("aws.region", "APP_AWS_REGION")
	viper.BindEnv("aws.access_key_id", "APP_AWS_ACCESS_KEY_ID")
	viper.BindEnv("aws.secret_access_key", "APP_AWS_SECRET_ACCESS_KEY")
	viper.BindEnv("aws.s3_bucket", "APP_AWS_S3_BUCKET")
	viper.BindEnv("aws.s3_host", "APP_AWS_S3_HOST")

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
