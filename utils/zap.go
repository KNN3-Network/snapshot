package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger

func init() {
	// 创建一个 lumberjack 实例，用于处理日志文件的切分和删除
	lj := &lumberjack.Logger{
		Filename:   "logger.log",
		MaxSize:    100, // 按大小切分，单位 MB
		MaxBackups: 0,
		MaxAge:     3,     // 保留最近 3 天的日志文件
		Compress:   false, // 是否启用压缩
	}

	ws := zapcore.AddSync(lj)

	// 创建一个 encoder 配置
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	consoleLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	})
	consoleCore := zapcore.NewCore(consoleEncoder, consoleDebugging, consoleLevel)

	fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
	fileLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	})
	fileCore := zapcore.NewCore(fileEncoder, ws, fileLevel)

	// 创建一个新的 Core，将两个子 Core 组合起来
	core := zapcore.NewTee(consoleCore, fileCore)

	// 创建 logger
	Logger = zap.New(core)
}
