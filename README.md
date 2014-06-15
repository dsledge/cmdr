cmdr
====

Go command execution runner for local or remote ssh commands

###Running Tests:
```bash
#username and password authentication (Runs an "ls -la /" command)
go test -v -username <username> -password <password> -sshserver <ip addr:port>

#username and pemfile authentication (Runs an "ls -la" command)
go test -v -username <username> -pemfile <pemfile> -sshserver <ip addr:port>
```

###Local Command Example:
```go
package main

import (
  "github.com/dsledge/cmdr"
  "fmt"
)

func main() {
  output := make(chan string)
  errout := make(chan string)
  
  cmd, err := cmdr.NewCommand(nil, output, errout)
  if err != nil {
    return err
  }
  
  go func() {
    for {
      select {
      case out := <-output:
        fmt.Printf("%s", out)
      case err := <-errout:
        fmt.Printf("%s", err)
      }
    }
  }()
  
  err = cmd.Execute("ls", "-la", "/")
  if err != nil {
    fmt.Printf("%s", err)
  }
}
```

###Remote SSH Command Example:
```go
package main

import (
  "github.com/dsledge/cmdr"
  "fmt"
)

func main() {
  output := make(chan string)
  errout := make(chan string)
  
  config, err := cmdr.NewClientConfig(*username, *password, *pemfile)
  if err != nil {
    t.Errorf("%s\n", err)
  }
  
  cmd, err := cmdr.NewSSHCommand(config, "<ip address>:<port>", nil, output, errout)
  defer cmd.Close()
  if err != nil {
    t.Errorf("%s\n", err)
  }
  
  go func() {
    for {
      select {
      case out := <-output:
        fmt.Printf("%s", out)
      case err := <-errout:
        fmt.Printf("%s", err)
      }
    }
  }()
  
  err = cmd.Execute("ls -la")
  if err != nil {
    fmt.Printf("%s", err)
  }
}
```
