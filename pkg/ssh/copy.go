package ssh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func (s *SSH) Copy(localPath, remotePath string) error {
	// if s.isLocalAction(host) {
	// 	logger.Debug("local %s copy files src %s to dst %s", host, localPath, remotePath)
	// 	return file.RecursionCopy(localPath, remotePath)
	// }
	s.Infof("remote copy files src %s to dst %s:%s", localPath, s.Address, remotePath)
	sshClient, sftpClient, err := s.sftpConnectWithRetry()
	if err != nil {
		return fmt.Errorf("failed to connect: %s", err)
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()

	f, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("get file stat failed %s", err)
	}
	rfp, err := sftpClient.Stat(remotePath)
	if err == nil && !f.IsDir() && rfp.IsDir() {
		log.Println("copy file to remote dir")
		remotePath = filepath.Join(remotePath, filepath.Base(localPath))
	}
	remoteDir := filepath.Dir(remotePath)
	rfp, err = sftpClient.Stat(remoteDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = sftpClient.MkdirAll(remoteDir); err != nil {
			return fmt.Errorf("failed to Mkdir remote: %v", err)
		}
	} else if !rfp.IsDir() {
		return fmt.Errorf("dir of remote file %s is not a directory", remotePath)
	}
	number := 1
	if f.IsDir() {
		number = CountDirFiles(localPath)
		// no files in local dir, but still need to create remote dir
		if number == 0 {
			return sftpClient.MkdirAll(remotePath)
		}
	}
	return s.doCopy(sftpClient, s.Address, localPath, remotePath)
}
func (s *SSH) doCopy(client *sftp.Client, host, src, dest string) error {
	lfp, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to Stat local: %v", err)
	}
	if lfp.IsDir() {
		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("failed to ReadDir: %v", err)
		}
		if err = client.MkdirAll(dest); err != nil {
			return fmt.Errorf("failed to Mkdir remote: %v", err)
		}
		for _, entry := range entries {
			if err = s.doCopy(client, host, path.Join(src, entry.Name()), path.Join(dest, entry.Name())); err != nil {
				return err
			}
		}
	} else {
		fn := func(host string, name string) bool {
			exists, err := checkIfRemoteFileExists(client, name)
			if err != nil {
				log.Printf("failed to detect remote file exists: %v", err)
			}
			return exists
		}
		if isEnvTrue("USE_SHELL_TO_CHECK_FILE_EXISTS") {
			fn = s.remoteFileExist
		}
		if !isEnvTrue("DO_NOT_CHECKSUM") && fn(host, dest) {
			rfp, _ := client.Stat(dest)
			if lfp.Size() == rfp.Size() && FileDigest(src) == s.RemoteSha256Sum(dest) {

				log.Printf("remote dst %s already exists and is the latest version, skip copying process", dest)
				return nil
			}
		}
		lf, err := os.Open(filepath.Clean(src))
		if err != nil {
			return fmt.Errorf("failed to open: %v", err)
		}
		defer lf.Close()

		dstfp, err := client.Create(dest)
		if err != nil {
			return fmt.Errorf("failed to create: %v", err)
		}
		if err = dstfp.Chmod(lfp.Mode()); err != nil {
			return fmt.Errorf("failed to Chmod dst: %v", err)
		}
		defer dstfp.Close()
		if _, err = io.Copy(dstfp, lf); err != nil {
			return fmt.Errorf("failed to Copy: %v", err)
		}
		if !isEnvTrue("DO_NOT_CHECKSUM") {
			dh := s.RemoteSha256Sum(dest)
			if dh == "" {
				// when ssh connection failed, remote sha256 is default to "", so ignore it.
				return nil
			}
			sh := FileDigest(src)
			if sh != dh {
				return fmt.Errorf("sha256 sum not match %s(%s) != %s(%s), maybe network corruption?", src, sh, dest, dh)
			}
		}
	}
	return nil
}
func (s *SSH) CmdToString(cmd, sep string) (string, error) {
	output, err := s.Cmd(cmd)
	data := string(output)
	if err != nil {
		return data, err
	}

	if len(data) == 0 {
		return "", fmt.Errorf("command %s on %s return nil", cmd, s.Address)
	}
	return getOnelineResult(data, sep), nil
}

func getOnelineResult(output string, sep string) string {
	return strings.ReplaceAll(strings.ReplaceAll(output, "\r\n", sep), "\n", sep)
}

func (s *SSH) sftpConnect() (*ssh.Client, *sftp.Client, error) {
	sshClient, err := s.connect()
	if err != nil {
		return nil, nil, err
	}
	// create sftp client
	sftpClient, err := sftp.NewClient(sshClient)
	return sshClient, sftpClient, err
}

func (s *SSH) sftpConnectWithRetry() (sshClient *ssh.Client, sftpClient *sftp.Client, err error) {
	for i := 0; i < defaultMaxRetry; i++ {
		if i > 0 {
			log.Printf("trying to reconnect due to error occur: %v", err)
			time.Sleep(time.Millisecond * 100)
		}
		sshClient, sftpClient, err = s.sftpConnect()
		if err == nil || !isErrorWorthRetry(err) {
			break
		}
	}
	return
}

func isErrorWorthRetry(err error) bool {
	return strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), io.EOF.Error())
}

func checkIfRemoteFileExists(client *sftp.Client, fp string) (bool, error) {
	_, err := client.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isEnvTrue(k string) bool {
	if v, ok := os.LookupEnv(k); ok {
		boolVal, _ := strconv.ParseBool(v)
		return boolVal
	}
	return false
}

func (s *SSH) RemoteSha256Sum(remoteFilePath string) string {
	cmd := fmt.Sprintf("sha256sum %s | cut -d\" \" -f1", remoteFilePath)
	remoteHash, err := s.CmdToString(cmd, "")
	if err != nil {
		log.Printf("failed to calculate remote sha256 sum %s %s %v", s.Address, remoteFilePath, err)
	}

	return remoteHash
}

func (s *SSH) remoteFileExist(host, remoteFilePath string) bool {
	// if remote file is
	// ls -l | grep aa | wc -l
	remoteFileName := path.Base(remoteFilePath) // aa
	remoteFileDirName := path.Dir(remoteFilePath)
	//it's bug: if file is aa.bak, `ls -l | grep aa | wc -l` is 1 ,should use `ll aa 2>/dev/null |wc -l`
	//remoteFileCommand := fmt.Sprintf("ls -l %s| grep %s | grep -v grep |wc -l", remoteFileDirName, remoteFileName)
	remoteFileCommand := fmt.Sprintf("ls -l %s/%s 2>/dev/null |wc -l", remoteFileDirName, remoteFileName)

	data, err := s.CmdToString(remoteFileCommand, " ")
	if err != nil {
		log.Printf("[ssh][%s]remoteFileCommand err:%s", host, err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(data))
	if err != nil {
		log.Printf("[ssh][%s]RemoteFileExist:%s", host, err)
	}
	return count != 0
}

func Digest(body []byte) string {
	bytes := sha256.Sum256(body)
	return hex.EncodeToString(bytes[:])
}

// FileDigest generates the sha256 digest of a file.
func FileDigest(path string) string {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		log.Printf("get file digest failed %v", err)
		return ""
	}

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Printf("get file digest failed %v", err)
		return ""
	}

	fileDigest := fmt.Sprintf("%x", h.Sum(nil))
	return fileDigest
}

func CountDirFiles(dirName string) int {
	if !IsDir(dirName) {
		return 0
	}
	var count int
	err := filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		log.Printf("count dir files failed %v", err)
		return 0
	}
	return count
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}
