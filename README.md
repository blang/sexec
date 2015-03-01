sexec - Simple command execution library for golang [![Build Status](https://drone.io/github.com/blang/sexec/status.png)](https://drone.io/github.com/blang/sexec/latest) [![GoDoc](https://godoc.org/github.com/blang/sexec?status.png)](https://godoc.org/github.com/blang/sexec) [![Coverage Status](https://img.shields.io/coveralls/blang/sexec.svg)](https://coveralls.io/r/blang/sexec?branch=master)
======

sexec is a simple command execution library written in golang.
It currently only works on linux.

Usage
-----
```bash
$ go get github.com/blang/sexec
```
Note: Always vendor your dependencies or fix on a specific version tag.

```go
import github.com/blang/sexec
p := sexec.NewProcess("while true; do echo test; sleep 1; done", os.Stdout, os.Stderr)
p.Start() // Start in background
p.Signal(syscall.SIGTERM) // Send signals
<- p.WaitCh() // Wait for exit
pid, err:=p.Pid()
code, err:=p.ExitCode()
```

Also check the [GoDocs](http://godoc.org/github.com/blang/sexec).

Why should I use this lib?
-----

- Simple
- Fully tested (Coverage >90%)
- Readable errors
- Only Stdlib


Features
-----

- Exit Codes
- Waiting via channel or method
- Signalling


Motivation
-----

I needed a simple lib to get ExitCodes and proper Signalling without the hassle around syscall.


Contribution
-----

Feel free to make a pull request. For bigger changes create a issue first to discuss about it.


License
-----

See [LICENSE](LICENSE) file.
