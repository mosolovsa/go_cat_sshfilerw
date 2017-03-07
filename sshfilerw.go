package go_cat_sshfilerw

import (
	"golang.org/x/crypto/ssh"
	"errors"
	"fmt"
	"io"
	"runtime"
)

type SSHClient struct {
	addr   string
	cltcfg *ssh.ClientConfig
	clt    *ssh.Client
}

func NewSSHclt(addr string, cltcfg *ssh.ClientConfig) (*SSHClient, error) {
	c, err := ssh.Dial("tcp", addr, cltcfg)
	if err != nil {
		return nil, err
	}

	retval := SSHClient{
		addr:   addr,
		cltcfg: cltcfg,
		clt:    c,
	}
	//This not mean that you can forget about manual closing
	runtime.SetFinalizer(&retval, func(d interface{}) {
		dis, found := d.(*SSHClient)
		if found {
			dis.Close()
		}
	})

	return &retval, nil
}

func (c *SSHClient) Close() {
	c.clt.Close()
}

//Perform single command by SSH
//sshstdout io.Writer - interface of an instance for writing ssh output
func (c *SSHClient) Run(sshstdout io.Writer, cmd string) error {
	return c.perform(func(s *ssh.Session) error {
		if sshstdout != nil {
			s.Stderr = sshstdout
			s.Stdout = sshstdout
		}
		return s.Run(cmd)
	})
}

//Read file from the remote server
//wto io.Writer - interface of an instance, where you want to store the content of remote file
//rpath string - path to the file on the server
func (c *SSHClient) ReadFile(wto io.Writer, rpath string) error {
	return c.perform(func(s *ssh.Session) error {
		if wto == nil {
			return errors.New("Writer to write file content is not provided")
		}
		s.Stdout = wto
		return s.Run(fmt.Sprintf("cat %s", rpath))
	})
}

//Write file to the remote server
//rfrom io.Reader - interface of an instance with content, that should be written
//rpath string - path to the file on the server
func (c *SSHClient) WriteFile(rfrom io.Reader, rpath string) error {
	//perform cat stdin to the file, after perform return value session will be closed,
	// so the cat will be performed
	err := c.perform(func(s *ssh.Session) error {
		if rfrom == nil {
			return errors.New("Reader to read file content is not provided")
		}
		sshstdinPipe, err := s.StdinPipe()
		if err != nil {
			return err
		}

		done := make(chan error)
		go func(done chan error) {
			err = s.Start(fmt.Sprintf("cat > %s", rpath))
			done <- err
		}(done)
		err = <-done
		if err != nil {
			return err
		}

		_, err = io.Copy(sshstdinPipe, rfrom)
		if err != nil {
			return err
		}
		//cat waits for the newline symbol from stdin to perform writing
		_, err = sshstdinPipe.Write([]byte("\n"))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	//cat will perform write on receiving of '\n' byte, and write it to the file
	//here we cut off that last byte
	err = c.perform(func(s *ssh.Session) error {
		return s.Run(fmt.Sprintf("truncate --size=-1 %s", rpath))
	})
	return err

}

type operation func(s *ssh.Session) error

//Helper function, responsible for openning and closing ssh session. Session performs single command.
//As an argument must be passed a function, that should be performed
func (c *SSHClient) perform(op operation) error {
	s, err := c.clt.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()

	err = op(s)

	return err
}
