package logs

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
)

type Logger struct {
	// *zap.SugaredLogger
	*coreConfig
	cores cores
}
type cores []zapcore.Core

type coreConfig struct {
	logLevel zapcore.Level
	encoder  zapcore.Encoder
	logPath  string
}
type coreConfigFunc func(*coreConfig)

func WithLogLevel(logLevel zapcore.Level) coreConfigFunc {
	return func(c *coreConfig) {
		c.logLevel = logLevel
	}
}

func WithLogPath(logpath string) coreConfigFunc {
	return func(c *coreConfig) {
		c.logPath = logpath
	}
}
func WithJsonEncoder() coreConfigFunc {
	return func(c *coreConfig) {
		enc := zap.NewProductionEncoderConfig()
		enc.EncodeTime = customTimeEncoder
		c.encoder = zapcore.NewJSONEncoder(enc)
	}
}

func (l *Logger) NewCore(ccfs ...coreConfigFunc) *Logger {
	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = customTimeEncoder
	c := &coreConfig{encoder: zapcore.NewConsoleEncoder(enc)}
	var iw = os.Stdout
	for _, ccf := range ccfs {
		ccf(c)
	}
	if c.logPath != "" {
		f, err := os.OpenFile(c.logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend)
		if err != nil {
			panic(err)
		}
		iw = f
	}
	l.cores = append(l.cores,
		zapcore.NewCore(c.encoder, zapcore.AddSync(iw), c.logLevel))
	return l
}
func (l *Logger) Sugar() *zap.SugaredLogger {
	if len(l.cores) == 0 {
		l.NewCore()
	}
	return zap.New(zapcore.NewTee(l.cores...)).Sugar()
}

func NewLogger() *Logger {
	return &Logger{}
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.DateTime))
}
