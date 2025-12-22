package config

import (
	"encoding/json"
	"fmt"
	"os"

	"backend/model"

	"github.com/joho/godotenv"
)

type SystemConfigs struct {
	Config *model.EnvConfig
}

// LoadConfigs acts as your @PostConstruct
func LoadConfigs() (*SystemConfigs, error) {
	// 1. Read the 'config' env variable
	godotenv.Load()

	rawJson := os.Getenv("config")
	if rawJson == "" {
		return nil, fmt.Errorf("environment variable 'config' is empty or not set")
	}

	// 2. Parse JSON into the struct
	var envCfg model.EnvConfig
	err := json.Unmarshal([]byte(rawJson), &envCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &SystemConfigs{
		Config: &envCfg,
	}, nil
}
