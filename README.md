cmdr
====

Go command execution runner for local or remote ssh commands

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
  signer, err := cmdr.LoadPEM("file.pem")
  if err != nil {
    fmt.Errorf("%s", err)
  }
  
  config := &ssh.ClientConfig{
      User: "<username>",
      Auth: []ssh.AuthMethod{
              ssh.PublicKeys(signer),
      },
  }
  
  output := make(chan string)
  errout := make(chan string)
  
  cmd, err := NewSSHCommand(config, "<ip address>:<port>", nil, output, errout)
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
  }()
  
  err = cmd.Execute("ls", "-la")
  if err != nil {
    fmt.Printf("%s", err)
  }
}
```
