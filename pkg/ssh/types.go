package ssh

import (
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type Host struct {
	Address        string `yaml:"address,omitempty" json:"address,omitempty"`
	Port           int    `yaml:"port,omitempty" json:"port,omitempty"`
	User           string `yaml:"user,omitempty" json:"user,omitempty"`
	Password       string `yaml:"password,omitempty" json:"password,omitempty"`
	PrivateKey     string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	PrivateKeyPath string `yaml:"privateKeyPath,omitempty" json:"privateKeyPath,omitempty"`
}

const (
	defaultTimeout  = time.Duration(1) * time.Second
	defaultUser     = "root"
	defaultMaxRetry = 3
	Interactive     = true
	NoInteractive   = false
)

type SSH struct {
	*Host
	Stdout      bool
	Interactive bool
	Timeout     time.Duration
	// private properties
	clientConfig *ssh.ClientConfig
	*zap.SugaredLogger
}

// type Interface interface {
// 	// Copy is copy local files to remote host
// 	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
// 	// need check md5sum
// 	// Copy(host, srcFilePath, dstFilePath string) error
// 	// Cmd is exec command on remote host, and return combined standard output and standard error
// 	Cmd(cmd string) ([]byte, error)
// 	CmdS(cmd, logfilePath string) error
// 	Ping() error
// 	Copy(localPath, remotePath string) error
// }
