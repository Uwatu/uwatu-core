package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// Config holds all environment variables required by uwatu-core.
type Config struct {
	NokiaClientSecret   string `mapstructure:"NOKIA_CLIENT_SECRET"`
	NokiaBaseURL        string `mapstructure:"NOKIA_BASE_URL"`
	ATApiKey            string `mapstructure:"AT_API_KEY"`
	ATSandboxUsername   string `mapstructure:"AT_SANDBOX_USERNAME"`
	FirebaseProjectID   string `mapstructure:"FIREBASE_PROJECT_ID"`
	InternalIntelSecret string `mapstructure:"INTERNAL_INTEL_SECRET"`
	DatabaseURL         string `mapstructure:"DATABASE_URL"`
	JWTSecret           string `mapstructure:"JWT_SECRET"`
}

// MaskSecret masks all but the last 4 characters of a sensitive string.
func MaskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}

// LoadConfig reads configuration from the .env file or system environment variables.
func LoadConfig(path string) (Config, error) {
	var config Config

	viper.AddConfigPath(path)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}

	// FAIL-FAST VALIDATION
	if config.DatabaseURL == "" {
		return config, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	if config.ATApiKey == "" {
		return config, fmt.Errorf("AT_API_KEY environment variable is required")
	}
	if config.JWTSecret == "" {
		return config, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if config.NokiaClientSecret == "" {
		return config, fmt.Errorf("NOKIA_CLIENT_SECRET environment variable is required")
	}

	return config, nil
}
