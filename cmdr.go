package cmdr 

import (
	"code.google.com/p/go.crypto/ssh"
	"io"
	"io/ioutil"
	"bufio"
	"os/exec"
	"fmt"
	"reflect"
	"errors"
	"strings"
	"time"
)

func NewClientConfig(username, password, pemfile string) (*ssh.ClientConfig, error) {
	if username != "" && password != "" {
		answers := keyboardInteractive(map[string]string{"Password: ": password,})
		return &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.KeyboardInteractive(answers.Challenge),
			},
		}, nil
	}

	if username != "" && pemfile != "" {
		signer, err := loadPEM(pemfile)
		if err != nil {
			return nil, err
		}

		return &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}, nil
	}

	return nil, fmt.Errorf("Missing valid arguments, must pass a (username and password) or (username and pemfile).")
}

type Command struct {
	Session *exec.Cmd
	Stdin	chan string
	Stdout	chan string
	Stderr	chan string
	ErrDelimiter byte
	OutDelimiter byte
	ErrMaxBytes int
	OutMaxBytes int
	timeout		time.Duration
}

type SSHCommand struct {
	Command
	Config *ssh.ClientConfig
	Server string
	Session *ssh.Session
	client *ssh.Client
	timeout		time.Duration
}

func NewCommand(inchan, outchan, errchan chan string) (*Command, error) {
	return &Command{Stdin: inchan, Stdout: outchan, Stderr: errchan, timeout: 0}, nil
}

func NewSSHCommand(cfg *ssh.ClientConfig, server string, inchan, outchan, errchan chan string) (*SSHCommand, error) {
	return &SSHCommand{Config: cfg, Server: server, Command: Command{Stdin: inchan, Stdout: outchan, Stderr: errchan}, timeout: 0}, nil
}

func timeout(cmd *exec.Cmd, timeout time.Duration, done chan bool, err chan error) {
	if timeout > 0 {
		response := ""
		for {
			select {
			case <-time.After(timeout):
				cmd.Process.Kill()
				response = "Local command execution timeout reached, terminating command"
			case <-done:
				if len(response) > 1 {
					err <-fmt.Errorf(response)
				} else {
					err <-nil
				}
				break
			}
		}
	} else {
		err <-nil
	}
}

func timeout_ssh(client *ssh.Client, timeout time.Duration, done chan bool, err chan error) {
	if timeout > 0 {
		response := ""
		for {
			select {
			case <-time.After(timeout):
				client.Conn.Close()
				response = "Remote SSH command execution timeout reached, terminating command"
			case <-done:
				if len(response) > 1 {
					err <-fmt.Errorf(response)
				} else {
					err <-nil
				}
				break
			}
		}
	} else {
		err <-nil
	}
}

func (c *Command) Execute(cmd string, args ...string) error {
	c.Session = exec.Command(cmd, args...)
	done := make(chan bool, 1)
	errchan := make(chan error)
	defer close(done)
	go timeout(c.Session, c.timeout, done, errchan)

	if err := execute(c, ""); err != nil {
		fmt.Printf("Execute Error: %s\n", err)
		done <-true
		return err
	}

	done <-true
	err := <-errchan
	return err
}

func (s *SSHCommand) Execute(cmd string) (err error) {
	s.client, err = ssh.Dial("tcp", s.Server, s.Config)
	if err != nil {
		return err
	}
	s.Session, err = s.client.NewSession()
	if err != nil {
		return err
	}

	done := make(chan bool, 1)
	errchan := make(chan error)
	defer close(done)
	go timeout_ssh(s.client, s.timeout, done, errchan)

	if err = execute(s, cmd); err != nil {
		done <-true
		return err
	}

	done <-true
	err = <-errchan
	return err
}

func (c *Command) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (s *SSHCommand) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
}

func (c *Command) ProcessStdIn(notifier chan error, w io.WriteCloser) {
	processInput(c.Stdin, notifier, w)
}

func (c *Command) ProcessStdOut(notifier chan error, r io.Reader)  {
	processOutput(c.Stdout, notifier, r, c.OutMaxBytes, c.OutDelimiter)
}

func (c *Command) ProcessStdErr(notifier chan error, r io.Reader)  {
	processOutput(c.Stderr, notifier, r, c.ErrMaxBytes, c.ErrDelimiter)
}

func (c *SSHCommand) ProcessStdIn(notifier chan error, w io.WriteCloser) {
	processInput(c.Stdin, notifier, w)
}

func (s *SSHCommand) ProcessStdOut(notifier chan error, r io.Reader)  {
	processOutput(s.Stdout, notifier, r, s.OutMaxBytes, s.OutDelimiter)
}

func (s *SSHCommand) ProcessStdErr(notifier chan error, r io.Reader)  {
	processOutput(s.Stderr, notifier, r, s.ErrMaxBytes, s.ErrDelimiter)
}

func (s *SSHCommand) Close() {
	s.Session.Close()
	s.client.Close()
}

func execute(obj interface{}, cmd string) error {
	var innotifier chan error
	var outnotifier chan error
	var errnotifier chan error
	var ioerrs []string

	value := reflect.ValueOf(obj)
	vsession := value.Elem().FieldByName("Session")
	vstdin := value.Elem().FieldByName("Stdin")
	vstdout := value.Elem().FieldByName("Stdout")
	vstderr := value.Elem().FieldByName("Stderr")

	// Checking if a channel has been passed in to handle Stdout
	if !vstdin.IsNil() {
		innotifier = make(chan error)
		if method := vsession.MethodByName("StdinPipe"); method.IsValid() {
			values := method.Call(nil)
			if values[1].IsNil() {
				pipe := values[0].Interface()
				if processMethod := value.MethodByName("ProcessStdIn"); processMethod.IsValid() {
					go processMethod.Call([]reflect.Value{reflect.ValueOf(innotifier), reflect.ValueOf(pipe)})
				} else {
					return fmt.Errorf("ProcessStdIn method not found\n")
				}
			} else {
				return fmt.Errorf("An error occurred connecting up to Stdin: %s\n", values[1].Interface())
			}
		}
	}

	if !vstdout.IsNil() {
		outnotifier = make(chan error)
		if method := vsession.MethodByName("StdoutPipe"); method.IsValid() {
			values := method.Call(nil)
			if values[1].IsNil() {
				pipe := values[0].Interface()
				if processMethod := value.MethodByName("ProcessStdOut"); processMethod.IsValid() {
					go processMethod.Call([]reflect.Value{reflect.ValueOf(outnotifier), reflect.ValueOf(pipe)})
				} else {
					return fmt.Errorf("ProcessStdOut method not found\n")
				}
			} else {
				return fmt.Errorf("An error occurred connecting up to Stdout: %s\n", values[1].Interface())
			}
		}
	}
	if !vstderr.IsNil() {
		errnotifier = make(chan error)
		if method := vsession.MethodByName("StderrPipe"); method.IsValid() {
			values := method.Call(nil)
			if values[1].IsNil() {
				pipe := values[0].Interface()
				if processMethod := value.MethodByName("ProcessStdErr"); processMethod.IsValid() {
					go processMethod.Call([]reflect.Value{reflect.ValueOf(errnotifier), reflect.ValueOf(pipe)})
				} else {
					return fmt.Errorf("ProcessStdOut method not found\n")
				}
			} else {
				return fmt.Errorf("An error occurred connecting up to Stderr: %s\n", values[1].Interface())
			}
		}
	}

	// Run the command for the session
	if vstart := vsession.MethodByName("Start"); vstart.IsValid() {
		switch v := obj.(type) {
		case *Command:
			vstart.Call(nil)
		case *SSHCommand:
			vstart.Call([]reflect.Value{reflect.ValueOf(cmd)})
		default:
			return fmt.Errorf("Not a valid type, expected *Command or *SSHCommand but recevied %s", v)
		}
	}

	//Append stdin error if available
	if !vstdin.IsNil() {
		ioerrs = append(ioerrs, processErrors(innotifier)...)
	}

	//Append stdout errors if available
	if !vstdout.IsNil() {
		ioerrs = append(ioerrs, processErrors(outnotifier)...)
	}

	//Append stderr errors if available
	if !vstderr.IsNil() {
		ioerrs = append(ioerrs, processErrors(errnotifier)...)
	}

	//Iterate the errors and return them
	if ioerrs != nil && len(ioerrs) > 0 {
		errstr := "Errors found processing IO streams: \n"
		for i := 0; i < len(ioerrs); i++ {
			errstr = errstr + ioerrs[i]
		}
		return errors.New(errstr)
	}

	if vwait := vsession.MethodByName("Wait"); vwait.IsValid() {
		switch v := obj.(type) {
		case *Command:
			vwait.Call(nil)
		case *SSHCommand:
			vwait.Call(nil)
		default:
			return fmt.Errorf("Not a valid type, expected *Command or *SSHCommand but recevied %s", v)
		}
	}

	return nil
}

func processInput(in chan string, notifier chan error, w io.WriteCloser) {
	defer close(notifier)

	for {
		if in, ok := <-in; ok {
			input := strings.NewReader(in)
			if _, err := io.Copy(w, input); err != nil {
				notifier <-err
			}
		} else {
			return
		}
	}
}

func processOutput(out chan string, notifier chan error, r io.Reader, bytes int, delim byte) {
	defer close(notifier)
	defer close(out)

	bufr := bufio.NewReader(r)
	var str string
	var err error
	for {
		if bytes > 0 {
			var l int
			chars := make([]byte, bytes, bytes)
			l, err = io.ReadAtLeast(r,chars,bytes)
			str = string(chars[:l])
		} else {
			if delim == '\x00' {
				delim = '\n'
			}
			str, err = bufr.ReadString(delim)
		}
		if len(str) > 1 {
			out <-str
		}
		if err != nil {
			break
		}
	}
	if !(err == io.EOF || err == io.ErrUnexpectedEOF) {
		notifier <-err
	}
}

func processErrors(notifier chan error) []string {
	var errlist []string
	for {
		err, ok := <-notifier
		if !ok {
			return errlist
		}
		errlist = append(errlist, err.Error())
	}
}

func loadPEM(filename string) (ssh.Signer, error) {
	privateKey, _ := ioutil.ReadFile(filename)
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

type keyboardInteractive map[string]string

func (k *keyboardInteractive) Challenge(user, instruction string, questions []string, echos []bool) ([]string, error) {
	var answers []string
	for _, q := range questions {
		answers = append(answers, (*k)[q])
	}
	return answers, nil
}
