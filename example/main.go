package main

import (
	sshrw "github.com/mosolovsa/go_cat_sshfilerw"
	"log"
	"bytes"
	"bufio"
	"golang.org/x/crypto/ssh"
)

func main() {
	cfg := ssh.ClientConfig{
		User: "pi",
		Auth: []ssh.AuthMethod{
			ssh.Password("pass"),
		},
	}

	c, err := sshrw.NewSSHclt("192.168.0.11:22", &cfg)
	if err != nil {
		log.Panicln("Can't start ssh connection, err:", err.Error())
	}

	r := bytes.NewReader([]byte("abcd\n"))
	if err = c.WriteFile(r, "/home/pi/test"); err != nil {
		log.Println("Error on file write: ", err.Error())
	}

	var buff bytes.Buffer
	w := bufio.NewWriter(&buff)
	if err = c.ReadFile(w, "/home/pi/test"); err != nil {
		log.Println("Error on file read: ", err.Error())
	}
	w.Flush()
	log.Println(buff.String())
}
