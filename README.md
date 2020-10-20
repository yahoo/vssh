# vSSH

Go library to handle tens of thousands SSH connections and execute the command(s) with higher-level API for building network device / server automation. Documentation and examples are available via [godoc](https://godoc.org/github.com/yahoo/vssh).

[![Test Status](https://github.com/yahoo/vssh/workflows/vSSH/badge.svg)](https://github.com/yahoo/vssh/actions?query=workflow%3AvSSH)
[![Go Report Card](https://goreportcard.com/badge/github.com/yahoo/vssh)](https://goreportcard.com/report/github.com/yahoo/vssh)
[![Coverage Status](https://coveralls.io/repos/github/yahoo/vssh/badge.svg?branch=master&service=github1)](https://coveralls.io/github/yahoo/vssh?branch=master)
[![GoDoc](https://godoc.org/github.com/yahoo/vssh?status.svg)](https://godoc.org/github.com/yahoo/vssh)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/yahoo/vssh?tab=doc)](https://pkg.go.dev/github.com/yahoo/vssh?tab=doc)

![Alt text](/docs/imgs/vssh.png?raw=true "vSSH")

## Features
- Connect to multiple remote machines concurrently
- Persistent SSH connection
- DSL query based on the labels
- Manage number of sessions per SSH connection
- Limit amount of stdout and stderr data in bytes
- Higher-level API for building automation

### Sample query with label
```go
labels := map[string]string {
  "POP" : "LAX",
  "OS" : "JUNOS",
}
// sets labels to a client
vs.AddClient(addr, config, vssh.SetLabels(labels))
// query with label
vs.RunWithLabel(ctx, cmd, timeout, "(POP == LAX || POP == DCA) && OS == JUNOS")
```

### Basic example
```go
vs := vssh.New().Start()
config := vssh.GetConfigUserPass("vssh", "vssh")
for _, addr := range []string{"54.193.17.197:22", "192.168.2.19:22"} {
  vs.AddClient(addr, config, vssh.SetMaxSessions(4))
}
vs.Wait()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cmd:= "ping -c 4 192.168.55.10"
timeout, _ := time.ParseDuration("6s")
respChan := vs.Run(ctx, cmd, timeout)

for resp := range respChan {
  if err := resp.Err(); err != nil {
    log.Println(err)
      continue
    }

  outTxt, errTxt, _ := resp.GetText(vs)
  fmt.Println(outTxt, errTxt, resp.ExitStatus())
}
```

### Stream example
```go
vs := vssh.New().Start()
config, _ := vssh.GetConfigPEM("vssh", "mypem.pem")
vs.AddClient("54.193.17.197:22", config, vssh.SetMaxSessions(4))
vs.Wait()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cmd:= "ping -c 4 192.168.55.10"
timeout, _ := time.ParseDuration("6s")
respChan := vs.Run(ctx, cmd, timeout)

resp := <- respChan
if err := resp.Err(); err != nil {
  log.Fatal(err)
}

stream := resp.GetStream()
defer stream.Close()

for stream.ScanStdout() {
  txt := stream.TextStdout()
  fmt.Println(txt)
}
```
## Supported platform
- Linux
- Windows
- Darwin
- BSD
- Solaris

## License
Code is licensed under the Apache License, Version 2.0 (the "License").
Content is licensed under the CC BY 4.0 license. Terms available at https://creativecommons.org/licenses/by/4.0/.

## Contribute
Welcomes any kind of contribution, please follow the next steps:

- Fork the project on github.com.
- Create a new branch.
- Commit changes to the new branch.
- Send a pull request.
