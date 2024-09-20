package logs

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger(stdout, jsonOut bool, logfilePath string) *zap.Logger {

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = customTimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	if jsonOut {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	multiSyner := zapcore.NewMultiWriteSyncer(writeSyncers(stdout, logfilePath)...)
	core := zapcore.NewCore(encoder, multiSyner, zapcore.DebugLevel)
	return zap.New(core)
}

func writeSyncers(stdout bool, logfilePath string) (ws []zapcore.WriteSyncer) {
	if stdout {
		ws = append(ws, zapcore.AddSync(os.Stdout))
	}
	if logfilePath != "" {
		f, err := os.OpenFile(logfilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend)
		if err != nil {
			panic(err)
		}
		ws = append(ws, zapcore.AddSync(f))
	}
	return ws
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.DateTime))
}
