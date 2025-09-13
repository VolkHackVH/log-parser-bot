package logmonitor

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type LogMonitor struct {
	config    MonitorConfig
	session   *discordgo.Session
	logger    *zap.Logger
	filestate map[string]int64
}

func NewLogMonitor(session *discordgo.Session, logger *zap.Logger, config MonitorConfig) *LogMonitor {
	return &LogMonitor{
		config:    config,
		logger:    logger,
		session:   session,
		filestate: make(map[string]int64),
	}
}

func (lm *LogMonitor) Start() {
	lm.logger.Info("Starting log monitor",
		zap.Int("interval", lm.config.CheckInterval),
		zap.Int("log_count", len(lm.config.LogConfigs)))

	for _, cfg := range lm.config.LogConfigs {
		lm.logger.Info("Monitor log file",
			zap.String("name", cfg.LogName),
			zap.String("file", cfg.FilePath),
			zap.String("channel", cfg.FilePath))
	}

	ticker := time.NewTicker(time.Duration(lm.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lm.checkLogs()
	}
}

func (lm *LogMonitor) checkLogs() {
	for _, logConfig := range lm.config.LogConfigs {
		go lm.monitorSingleLog(logConfig)
	}
}

func (lm *LogMonitor) monitorSingleLog(config LogConfig) {
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		lm.logger.Debug("Log file does not exist",
			zap.String("file", config.FilePath),
			zap.String("name", config.LogName))
		return
	}

	file, err := os.Open(config.FilePath)
	if err != nil {
		lm.logger.Warn("Cannot open log file",
			zap.String("file", config.FilePath),
			zap.String("name", config.LogName),
			zap.Error(err))
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		lm.logger.Warn("Cannot get file info",
			zap.String("file", config.FilePath),
			zap.String("name", config.LogName),
			zap.Error(err))
		return
	}

	currentSize := fileInfo.Size()
	lastPosition := lm.filestate[config.FilePath]

	if currentSize < lastPosition {
		lastPosition = 0
	}

	if currentSize <= lastPosition {
		return
	}

	_, err = file.Seek(lastPosition, 0)
	if err != nil {
		lm.logger.Warn("Cannot seek file",
			zap.String("file", config.FilePath),
			zap.String("name", config.LogName),
			zap.Error(err))
		return
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		lm.sendLogLine(line, config)
	}

	lm.filestate[config.FilePath] = currentSize

	if lineCount > 0 {
		lm.logger.Debug("Sent new log lines",
			zap.String("name", config.LogName),
			zap.Int("count", lineCount))
	}

	if err := scanner.Err(); err != nil {
		lm.logger.Warn("Error reading log file",
			zap.String("file", config.FilePath),
			zap.String("name", config.LogName),
			zap.Error(err))
	}
}

func (lm *LogMonitor) sendLogLine(line string, config LogConfig) {
	message := fmt.Sprintf("`[%s]` %s", config.LogName, line)

	if len(message) > 2000 {
		message = message[:1997] + "..."
	}

	_, err := lm.session.ChannelMessageSend(config.ChannelID, message)
	if err != nil {
		lm.logger.Error("Error sending log message",
			zap.String("channel", config.ChannelID),
			zap.String("name", config.LogName),
			zap.Error(err))
	}
}
