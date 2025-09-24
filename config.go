package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 项目配置结构体
type Config struct {
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	UnitID         int    `json:"unitId"`
	SpeedDelays    []int  `json:"speedDelays"`
	PollIntervalMs int    `json:"pollIntervalMs"`
	WindowSize     []int  `json:"windowSize"`
	WindowPosition []int  `json:"windowPosition"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		IP:             "192.168.0.10",
		Port:           502,
		UnitID:         1,
		SpeedDelays:    []int{1000, 500, 200},
		PollIntervalMs: 200,
		WindowSize:     []int{800, 600},
		WindowPosition: []int{100, 100},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig() (*Config, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	configPath := filepath.Join(exPath, "config", "config.json")
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果不存在，创建默认配置
		return createDefaultConfig(configPath)
	}
	
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	// 解析配置
	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig(configPath string) (*Config, error) {
	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	
	// 获取默认配置
	config := DefaultConfig()
	
	// 保存默认配置到文件
	err := saveConfig(config, configPath)
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// saveConfig 保存配置到文件
func saveConfig(config *Config, configPath string) error {
	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	// 写入文件
	return os.WriteFile(configPath, data, 0644)
}

// SaveConfig 保存配置到默认位置
func (c *Config) SaveConfig() error {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	configPath := filepath.Join(exPath, "config", "config.json")
	return saveConfig(c, configPath)
}