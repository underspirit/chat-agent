package logger

import (
	"game-logic-server/internal/config"
)

func Init(zap *config.Zap) (*XLogger, error) {
	return New(Options{
		Level:        zap.Level,
		Director:     zap.Director,
		LinkName:     zap.LinkName,
		ShowLine:     zap.ShowLine,
		EncodeLevel:  zap.EncodeLevel,
		LogInConsole: zap.LogInConsole,
		MaxAge:       zap.MaxAge,
		MaxSize:      zap.MaxSize,
		Compress:     zap.Compress,
		RotationTime: zap.RotationTime,
		MaxBackup:    zap.MaxBackup,
	})
}
