package logmonitor

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LogChannel    map[string]string `json:"log_channels"`
	LogFiles      map[string]string `json:"log_files"`
	CheckInterval int               `json:"check_interval"`
}

type LogConfig struct {
	LogName   string
	FilePath  string
	ChannelID string
}

type MonitorConfig struct {
	LogConfigs    []LogConfig
	CheckInterval int
}

func LoadConfig(configPath string) (*Config, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) ToMonitorConfig() MonitorConfig {
	var logConfig []LogConfig

	for logName, channelID := range c.LogChannel {
		if baseDir, exists := c.LogFiles[logName]; exists {
			var matches []string

			logMatches, _ := filepath.Glob(filepath.Join(baseDir, "*"+logName+"*.log"))

			txtMatches, _ := filepath.Glob(filepath.Join(baseDir, "*"+logName+"*.txt"))

			matches = append(matches, logMatches...)
			matches = append(matches, txtMatches...)

			if len(matches) > 0 {
				logConfig = append(logConfig, LogConfig{
					LogName:   logName,
					FilePath:  matches[0],
					ChannelID: channelID,
				})
			}
		}
	}

	return MonitorConfig{
		LogConfigs:    logConfig,
		CheckInterval: c.CheckInterval,
	}
}

func GetDefaultConfig() MonitorConfig {
	return MonitorConfig{
		CheckInterval: 5,
		LogConfigs: []LogConfig{
			{
				LogName:   "player-death-logging",
				FilePath:  "/home/srv/Zomboid/Lua/",
				ChannelID: "CHANNEL_ID",
			},
		},
	}
}
