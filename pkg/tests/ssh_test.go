package tests

import (
	"testing"

	"github.com/funcx27/go-libs/pkg/logs"
	"github.com/funcx27/go-libs/pkg/ssh"
)

func Test_SSH(t *testing.T) {

	a := ssh.NewSSHClient(
		ssh.Host{
			Address:  "172.16.100.101",
			User:     "root",
			Port:     22,
			Password: "1234-abcd",
		})
	a.SugaredLogger = logs.NewLogger().NewCore(logs.WithLogPath("/tmp/ssh.log")).Sugar()
	a.CmdStream("echo 111111111;sleep 1;echo 2;sleep 1")
}
