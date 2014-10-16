package cmdr 

import (
	"testing"
	"time"
	"flag"
)

var pemfile = flag.String("pemfile","","The pemfile needed for ssh authentication")
var username = flag.String("username", "", "The username needed for ssh authentication")
var password = flag.String("password", "", "The password needed for ssh authentication")
var sshserver = flag.String("sshserver", "", "The ssh server to connect to, expecting <ip addr>:<port>")

func init() {
	flag.Parse()
}

func TestCommand(t *testing.T) {
	// t.Skip()
	//input := make(chan string)
	output := make(chan string)
	errout := make(chan string)

	cmd, err := NewCommand(nil, output, errout)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	go func() {
		var out string
		var err string
		for {
			select {
			case out = <-output:
				if len(out) > 0 {
					t.Logf("OUTPUT: %s", out)
				}
			case err = <-errout:
				if len(err) > 0 {
					t.Logf("ERROR: %s", err)
				}
			case <-time.After(10 * time.Second):
				t.Logf("Breaking infinite read loop because of timeout")
				return
			}
		}
	}()

	err = cmd.Execute("ls", "-la", "/")
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestCommandTimeout5Seconds(t *testing.T) {
	// t.Skip()
	//input := make(chan string)
	output := make(chan string)
	errout := make(chan string)

	cmd, err := NewCommand(nil, output, errout)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	go func() {
		var out string
		var err string
		for {
			select {
			case out = <-output:
				if len(out) > 0 {
					t.Logf("OUTPUT: %s", out)
				}
			case err = <-errout:
				if len(err) > 0 {
					t.Logf("ERROR: %s", err)
				}
			case <-time.After(30 * time.Second):
				// t.Logf("Breaking infinite read loop because of timeout")
				return
			}
		}
	}()

	cmd.SetTimeout(5*time.Second)
	err = cmd.Execute("sleep", "30")
	if err.Error() != "Local command execution timeout reached, terminating command" {
		t.Errorf("Expected response 'Local command execution timeout reached, terminating command' was not returned")
	}
}

func TestCommandTimeout0Seconds(t *testing.T) {
	t.Skip()
	//input := make(chan string)
	output := make(chan string)
	errout := make(chan string)

	cmd, err := NewCommand(nil, output, errout)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	go func() {
		var out string
		var err string
		for {
			select {
			case out = <-output:
				if len(out) > 0 {
					t.Logf("OUTPUT: %s", out)
				}
			case err = <-errout:
				if len(err) > 0 {
					t.Logf("ERROR: %s", err)
				}
			case <-time.After(30 * time.Second):
				// t.Logf("Breaking infinite read loop because of timeout")
				return
			}
		}
	}()

	err = cmd.Execute("sleep", "30")
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestSSHCommand(t *testing.T) {
	// t.Skip()
	// flag.Parse()

	config, err := NewClientConfig(*username, *password, *pemfile)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	//input := make(chan string)
	output := make(chan string)
	errout := make(chan string)

	sshcmd, err := NewSSHCommand(config, *sshserver, nil, output, errout)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	go func() {
		var out string
		var err string
		for {
			select {
			case out = <-output:
				if len(out) > 0 {
					t.Logf("OUTPUT: %s", out)
				}
			case err = <-errout:
				if len(err) > 0 {
					t.Logf("ERROR: %s", err)
				}
			case <-time.After(10 * time.Second):
				t.Logf("Breaking infinite read loop because of timeout")
				return
			}
		}
	}()

	err = sshcmd.Execute("uptime")
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestSSHCommandTimeout(t *testing.T) {
	// t.Skip()
	// flag.Parse()

	config, err := NewClientConfig(*username, *password, *pemfile)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	//input := make(chan string)
	output := make(chan string)
	errout := make(chan string)

	sshcmd, err := NewSSHCommand(config, *sshserver, nil, output, errout)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	go func() {
		var out string
		var err string
		for {
			select {
			case out = <-output:
				if len(out) > 0 {
					t.Logf("OUTPUT: %s", out)
				}
			case err = <-errout:
				if len(err) > 0 {
					t.Logf("ERROR: %s", err)
				}
			case <-time.After(10 * time.Second):
				t.Logf("Breaking infinite read loop because of timeout")
				return
			}
		}
	}()

	sshcmd.SetTimeout(5*time.Second)
	err = sshcmd.Execute("sleep 20")
	if err.Error() != "Remote SSH command execution timeout reached, terminating command" {
		t.Errorf("Expected response 'Remote SSH command execution timeout reached, terminating command' was not returned")
	}
}
