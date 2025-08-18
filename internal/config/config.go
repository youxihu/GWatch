// internal/config/config.go
package config

import (
	"GWatch/internal/entity"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var GlobalConfig *entity.Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*entity.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config entity.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	GlobalConfig = &config
	return &config, nil
}

// GetConfig 获取全局配置
func GetConfig() *entity.Config {
	return GlobalConfig
} 