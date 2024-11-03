package tests

import (
	"testing"

	"github.com/funcx27/go-libs/pkg/logs"
)

func Test_Logger(t *testing.T) {
	log1 := logs.NewLogger().NewCore(logs.WithJsonEncoder()).Sugar()
	log1.Info("test1112")
	// log2.Info("333")
}
