package ssh

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func (s *SSH) Session() (*ssh.Client, *ssh.Session, error) {
	client, err := s.connect()
	if err != nil {
		return nil, nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, nil, err
	}
	return client, session, nil
}

func (s *SSH) connect() (*ssh.Client, error) {
	if s.clientConfig == nil {
		config := ssh.Config{
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
		}
		if s.Timeout <= 0 {
			s.Timeout = defaultTimeout
		}
		if s.User == "" {
			s.User = defaultUser
		}
		s.clientConfig = &ssh.ClientConfig{
			User:    s.User,
			Auth:    s.sshAuthMethod(),
			Timeout: s.Timeout,
			Config:  config,
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}
	}
	addr := addrReformat(s.Address, s.Port)
	return ssh.Dial("tcp", addr, s.clientConfig)
}

func (h *Host) sshAuthMethod() (auth []ssh.AuthMethod) {
	if h.PrivateKey != "" {
		signer, err := parsePrivateKey([]byte(h.PrivateKey), []byte(h.Password))
		if err == nil {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}
	if fileExist(h.PrivateKeyPath) {
		signer, err := parsePrivateKeyFile(h.PrivateKeyPath, h.Password)
		if err == nil {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}
	if h.Password != "" {
		auth = append(auth, ssh.Password(h.Password))
	}
	return auth
}

func parsePrivateKey(pemBytes []byte, password []byte) (ssh.Signer, error) {
	if len(password) == 0 {
		return ssh.ParsePrivateKey(pemBytes)
	}
	return ssh.ParsePrivateKeyWithPassphrase(pemBytes, password)
}

func parsePrivateKeyFile(filename string, password string) (ssh.Signer, error) {
	pemBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %v", err)
	}
	return parsePrivateKey(pemBytes, []byte(password))
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func addrReformat(host string, port int) string {
	if port == 0 {
		port = 22
	}
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, port)
	}
	return host
}

func interactiveSession(session *ssh.Session) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     //disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	w, h := termSize()
	if err := session.RequestPty("xterm", h, w, modes); err != nil {
		_ = session.Close()
		return err
	}
	return nil
}

func termSize() (w, h int) {
	w, h = 80, 24
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		log.Printf("terminal make raw: %s", err)
	}
	defer term.Restore(fd, state)
	w, h, err = term.GetSize(fd)
	if err != nil {
		log.Printf("terminal get size: %s", err)
	}
	return
}
