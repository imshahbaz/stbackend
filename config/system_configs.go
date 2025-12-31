package config

import (
	"backend/model"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
)

type SystemConfigs struct {
	Config *model.EnvConfig
}

func LoadConfigs() (*SystemConfigs, error) {
	godotenv.Load()

	rawJson := os.Getenv("config")
	if rawJson == "" {
		return nil, fmt.Errorf("environment variable 'config' is empty or not set")
	}

	var envCfg model.EnvConfig
	err := json.Unmarshal([]byte(rawJson), &envCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &SystemConfigs{
		Config: &envCfg,
	}, nil
}

type ConfigManager struct {
	value atomic.Value
}

func NewConfigManager(initial *model.MongoEnvConfig) *ConfigManager {
	cm := &ConfigManager{}
	cm.value.Store(initial)
	return cm
}

func (cm *ConfigManager) GetConfig() *model.MongoEnvConfig {
	return cm.value.Load().(*model.MongoEnvConfig)
}

func (cm *ConfigManager) UpdateConfig(newCfg *model.MongoEnvConfig) {
	cm.value.Store(newCfg)
}
