package core

import (
	"bytes"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"runtime"
)

type SSHTool struct {
	Host     string
	Port     int
	User     string
	Password string

	sshClient *ssh.Client
}

func NewSSHTool(host string, port int, user string, pass string) (*SSHTool, error) {
	sshTool := &SSHTool{Host: host, Port: port, User: user, Password: pass}
	// 创建SSH客户端配置
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
			// 如果使用私钥认证，可以使用下面的代码替换上面的密码认证
			// ssh.PublicKeys(getPrivateKey("path/to/private/key")),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接到SSH服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %s", err)
	}

	sshTool.sshClient = client
	return sshTool, nil
}

func (sshTool *SSHTool) Upload(localPath string, remotePath string) error {
	sftpClient, err := sftp.NewClient(sshTool.sshClient)
	if err != nil {
		return err
	}
	defer Close(sftpClient)

	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %s", err)
	}
	defer Close(localFile)

	// 创建远程文件
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %s", err)
	}
	defer Close(remoteFile)

	// 复制文件内容
	buf, err := io.ReadAll(localFile)
	if err != nil {
		return fmt.Errorf("读取本地文件内容失败: %s", err)
	}

	if _, err := remoteFile.Write(buf); err != nil {
		return fmt.Errorf("写入远程文件失败: %s", err)
	}

	return err
}

func (sshTool *SSHTool) Download(remoteFile, localFile string) error {
	sftpClient, err := sftp.NewClient(sshTool.sshClient)
	if err != nil {
		return err
	}
	defer Close(sftpClient)

	sftpFile, err := sftpClient.Open(remoteFile)
	if err != nil {
		return err
	}
	defer Close(sftpFile)

	fInfo, err := sftpClient.Stat(remoteFile)
	if err != nil {
		return err
	}

	file, err := os.Create(localFile)
	if err != nil {
		return err
	}
	defer Close(file)

	size, err := io.Copy(file, sftpFile)
	if err != nil {
		return err
	}

	if fInfo.Size() != size {
		return fmt.Errorf("local size is not equal to remote size (%d != %d)", size, fInfo.Size())
	}
	return nil
}

func (sshTool *SSHTool) Exec(cmd string) (stdout string, stderr string, err error) {
	session, err := sshTool.sshClient.NewSession()
	if err != nil {
		err = fmt.Errorf("创建会话失败: %s", err)
		return "", "", err
	}
	//defer Close(session)  // session.Run 会关闭 session

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	session.Stderr = &stderrBuffer
	session.Stdout = &stdoutBuffer

	err = session.Run(cmd)
	stdout = stdoutBuffer.String()
	stderr = stderrBuffer.String()
	return
}

func (sshTool *SSHTool) PrintExec(cmd string) {
	log.Println("#", cmd)
	stdout, stderr, err := sshTool.Exec(cmd)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	log.Println(">", stdout)
	if stderr != "" {
		log.Println("x", stderr)
	}
}

func (sshTool *SSHTool) Close() error {
	return sshTool.sshClient.Close()
}

func Close(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("%s-%d关闭错误: %s", file, line, err)
	}
}
