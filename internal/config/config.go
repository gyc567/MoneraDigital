package config

import (
        "github.com/spf13/viper"
)

type Config struct {
        Port        string
        DatabaseURL string
        RedisURL    string
        JWTSecret   string
}

func Load() *Config {
        viper.SetConfigFile(".env")
        viper.ReadInConfig() // Ignore error if .env doesn't exist

        // Use PORT from environment, default to 80 for Cloud Run compatibility
        viper.SetDefault("PORT", "80")
        viper.SetDefault("DATABASE_URL", "postgres://user:password@localhost/monera?sslmode=disable")
        viper.SetDefault("REDIS_URL", "redis://localhost:6379")
        viper.SetDefault("JWT_SECRET", "your-secret-key")

        viper.AutomaticEnv()

        cfg := &Config{
                Port:        viper.GetString("PORT"),
                DatabaseURL: viper.GetString("DATABASE_URL"),
                RedisURL:    viper.GetString("REDIS_URL"),
                JWTSecret:   viper.GetString("JWT_SECRET"),
        }

        return cfg
}
