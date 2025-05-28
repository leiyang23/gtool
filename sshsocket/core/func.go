package core

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
)

func Conn(cli *ssh.Client) (stdin io.WriteCloser, err error) {
	session, err := cli.NewSession()
	if err != nil {
		return
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0, // 关闭回显
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = session.RequestPty("xterm", 70, 128, modes)
	if err != nil {
		return
	}

	stdin, err = session.StdinPipe()
	if err != nil {
		err = fmt.Errorf("error creating stdin pipe: %v", err)
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		err = fmt.Errorf("failed to create stdout pipe: %v", err)
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		err = fmt.Errorf("failed to create stderr pipe: %v", err)
		return
	}

	// 进入docker容器内
	dockerCmd := fmt.Sprintf("sudo docker exec -it ContainerId sh")
	fmt.Println(dockerCmd)
	//err = session.Start(dockerCmd)

	// 这是进入主机
	err = session.Shell()
	if err != nil {
		return
	}

	go func() {
		for {
			buf := make([]byte, 10240)
			n, err2 := stdout.Read(buf)
			if err2 != nil {
				if err2 != io.EOF {
					fmt.Println("读取数据error: ", err2)
					break
				}
			}

			fmt.Printf(string(buf[:n]))
		}
		fmt.Println("read err over")
	}()

	go func() {
		for {
			buf := make([]byte, 10240)
			n, err2 := stderr.Read(buf)
			if err2 != nil {
				if err2 != io.EOF {
					fmt.Println("读取数据error: ", err2)
					break
				}
			}

			fmt.Println(3333, string(buf[:n]))
		}

		fmt.Println("read err over")
	}()

	return
}

func getPrivateKey(pKeyPath string) (s ssh.Signer, err error) {

	// 加载私钥
	key, err := os.ReadFile(pKeyPath)
	if err != nil {
		err = fmt.Errorf("无法读取私钥文件: %v", err)
		return
	}
	// 解析私钥
	s, err = ssh.ParsePrivateKey(key)
	if err != nil {
		err = fmt.Errorf("无法解析私钥: %v", err)
		return
	}

	return
}

func GetSSHClient(ip string, port int, user, password string) (client *ssh.Client, err error) {
	var auth ssh.AuthMethod
	if password == "" {
		var s ssh.Signer
		s, err = getPrivateKey("/root/.ssh/id_rsa")
		if err != nil {
			return
		}
		auth = ssh.PublicKeys(s)
	} else {
		auth = ssh.Password(password)
	}

	client, err = ssh.Dial("tcp", fmt.Sprintf("%s:%v", ip, port), &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		err = fmt.Errorf("连接远程主机失败: %v", err)
		return
	}

	return
}
