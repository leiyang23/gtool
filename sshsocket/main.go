package main

import (
	"bufio"
	"gitee.com/code_butcher/gtool/sshsocket/core"
	"golang.org/x/crypto/ssh"
	"os"
)

var sshCli *ssh.Client

func init() {
	var err error
	sshCli, err = core.GetSSHClient("10.10.**.**", 22, "root", "****")
	if err != nil {
		panic(err)
	}
}

func main() {
	stdin, err := core.Conn(sshCli)
	if err != nil {
		panic(err)
	}

	var input string
	f := bufio.NewReader(os.Stdin)
	for {
		input, _ = f.ReadString('\n')
		_, err = stdin.Write([]byte(input + "\n"))
		if err != nil {
			panic(err)
		}
	}

	sshCli.Close()
}
