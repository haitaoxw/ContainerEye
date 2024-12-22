package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Database struct {
		Path string
	}
	Alert struct {
		Slack struct {
			Token   string
			Channel string
		}
		Email struct {
			SMTPHost    string
			SMTPPort    int
			From        string
			Password    string
			ToReceivers []string
		}
	}
	Server struct {
		Port int
	}
}

// LoadConfig loads the configuration from config.yaml
func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	var config Config

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use default values
			config.Database.Path = "data/containereye.db"
			config.Server.Port = 8080
			
			// Create default config file
			viper.Set("database.path", config.Database.Path)
			viper.Set("server.port", config.Server.Port)
			
			// Ensure data directory exists
			if err := os.MkdirAll("data", 0755); err != nil {
				fmt.Printf("Warning: Failed to create data directory: %v\n", err)
			}
			
			if err := viper.SafeWriteConfig(); err != nil {
				fmt.Printf("Warning: Failed to write default config: %v\n", err)
			}
		} else {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	} else {
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Printf("Error unmarshaling config: %v\n", err)
		}
	}

	return &config
}
