package main

import (
	"log"
	"os"
	"os/signal"
	"pd/cmd/commands"
	"pd/internal/logmonitor"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env not loaded!", err)
		return
	}

	logz := initLogger(true)
	defer logz.Sync()

	token := os.Getenv("TOKEN")
	if token == "" {
		logz.Fatal("Token bot is required!\nSet the TOKEN variable in your .env file or using an application.")
	}
	ds, err := discordgo.New("Bot " + token)
	if err != nil {
		logz.Panic("Error create session", zap.Error(err))
	}
	defer ds.Close()

	ds.AddHandler(commands.MessageCreate)

	if err := ds.Open(); err != nil {
		logz.Panic("Error opening connection: ", zap.Error(err))
	}

	logz.Info("Bot started and connected to Discord!")

	config, err := logmonitor.LoadConfig("config.json")
	if err != nil {
		logz.Warn("Cannot load config.json, using default config", zap.Error(err))
		defaultConfig := logmonitor.GetDefaultConfig()
		monitor := logmonitor.NewLogMonitor(ds, logz, defaultConfig)
		go monitor.Start()
	} else {
		for logName, channelID := range config.LogChannel {
			envChannelID := os.Getenv(channelID)
			if envChannelID != "" {
				config.LogChannel[logName] = envChannelID
			}
		}

		monitorConfig := config.ToMonitorConfig()
		monitor := logmonitor.NewLogMonitor(ds, logz, monitorConfig)
		go monitor.Start()
	}

	logz.Info("Log monitoring started!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	logz.Info("Ctrl+C to exit")
	<-sc

	logz.Info("Bot shutting down...")

}

func initLogger(debug bool) *zap.Logger {
	level := zap.InfoLevel
	if debug {
		level = zap.DebugLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalColorLevelEncoder,
		EncodeTime:    zapcore.TimeEncoderOfLayout("15:04:05.000"),
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)

	logger := zap.New(core, zap.AddCaller())
	return logger
}
