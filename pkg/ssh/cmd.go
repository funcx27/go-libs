package ssh

import (
	"bufio"
	"fmt"

	"github.com/funcx27/go-libs/pkg/logs"
)

func NewSSHClient(host Host) *SSH {
	return &SSH{
		Host:          &host,
		Stdout:        true,
		Interactive:   false,
		SugaredLogger: logs.NewLogger().Sugar(),
	}
}

func (s *SSH) Ping() error {
	client, _, err := s.Session()
	if err != nil {
		return fmt.Errorf("[ssh %s]create ssh session failed, %v", s.Address, err)
	}
	return client.Close()
}
func (s *SSH) Cmd(cmd string) ([]byte, error) {
	_, session, err := s.Session()
	if err != nil {
		return nil, err
	}
	b, err := session.CombinedOutput(cmd)
	defer session.Close()
	if s.Stdout {
		fmt.Println(string(b))
	}
	return b, err
}

func (s *SSH) CmdStream(cmd string) error {
	client, session, err := s.Session()
	if err != nil {
		return err
	}
	defer session.Close()
	defer client.Close()
	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()
	session.Start(cmd)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		s.Info(scanner.Text())
	}
	scanner = bufio.NewScanner(stderr)
	for scanner.Scan() {
		s.Info(scanner.Text())
	}
	return session.Wait()
}
