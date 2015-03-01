package sexec

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

var (
	ErrorStillRunning   = errors.New("Process still running")
	ErrorNotRunning     = errors.New("Process not running")
	ErrorNotStarted     = errors.New("Process not started")
	ErrorAlreadyRunning = errors.New("Process already running")
)

func NewProcess(command string, stdout io.Writer, stderr io.Writer) *Process {
	return &Process{
		Command: command,
		Stdin:   nil,
		Stdout:  stdout,
		Stderr:  stderr,
	}
}

type Process struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	cmd     *exec.Cmd
	mon     *monitorStatus
}

type monitorStatus struct {
	exitCode int
	running  bool
	ch       chan struct{}
}

func (m monitorStatus) wait() {
	<-m.ch
}

func (p *Process) initMonitor() {
	p.mon = &monitorStatus{}
	p.mon.running = true
	p.mon.ch = make(chan struct{})
}

func (p *Process) closeMonitor() {
	p.mon.running = false
	close(p.mon.ch)
}

func (p *Process) monitor() {
	defer p.closeMonitor()
	if err := p.cmd.Wait(); err != nil {

		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					// Error code for signals
					p.mon.exitCode = 128 + int(status.Signal())
					return
				} else {
					p.mon.exitCode = status.ExitStatus()
					return
				}
			}
		}
		// Return general error
		p.mon.exitCode = 1
		return
	} else {
		// Get error from process state, should be 0
		waitstatus := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		p.mon.exitCode = waitstatus.ExitStatus()
		return
	}
}

// Run starts process and waits for it to complete
func (p *Process) Run() error {
	if p.mon != nil && p.mon.running {
		return ErrorAlreadyRunning
	}
	p.initMonitor()
	err := p.start()
	if err != nil {
		p.closeMonitor()
		return err
	}
	go p.monitor()
	p.mon.wait()

	return nil
}

// Start starts process without waiting for it to complete
func (p *Process) Start() error {
	p.initMonitor()
	err := p.start()
	if err != nil {
		p.closeMonitor()
		return err
	}
	go p.monitor()

	return nil
}

// start starts process without waiting for it to complete
func (p *Process) start() error {
	cmd := exec.Command("/bin/bash", "-c", p.Command)
	cmd.Stdin = p.Stdin
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	cmd.Env = os.Environ()
	p.cmd = cmd

	return cmd.Start()
}

// Pid returns the PID of the process. Pid remains after process exists.
func (p *Process) Pid() (int, error) {
	if p.mon == nil || p.cmd.Process == nil {
		return 0, ErrorNotRunning
	}
	return p.cmd.Process.Pid, nil
}

// Wait waits for process to complete, returns exitCode or error if process is not running.
func (p *Process) Wait() (int, error) {
	if p.mon == nil {
		return -1, ErrorNotRunning
	}
	p.mon.wait()
	return p.mon.exitCode, nil
}

// WaitCh waits for process to complete and closes returned channel.
func (p *Process) WaitCh() chan struct{} {
	if p.mon == nil {
		return nil
	}
	ch := make(chan struct{})
	go func() {
		<-p.mon.ch
		close(ch)
	}()
	return ch
}

// WaitCh waits for process to complete and closes given channel.
func (p *Process) WaitOnCh(ch chan struct{}) error {
	if p.mon == nil {
		return ErrorNotRunning
	}
	if ch == nil {
		panic("Nil channel")
	}

	go func() {
		<-p.mon.ch
		close(ch)
	}()
	return nil
}

// ExitCode returns the process' exitcode or error if not started or still running.
func (p *Process) ExitCode() (int, error) {
	if p.mon == nil {
		return -1, ErrorNotStarted
	}
	if p.mon.running {
		return -1, ErrorStillRunning
	}
	return p.mon.exitCode, nil
}

// Success returns true iff process returned with exitcode 0.
func (p *Process) Success() bool {
	if p.mon == nil || p.mon.running {
		return false
	}
	return p.mon.exitCode == 0
}

// Signal sends a signal to the process.
func (p *Process) Signal(sig os.Signal) error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Signal(sig)
	} else {
		return ErrorNotRunning
	}
}

// Started returns true iff process has been started before.
func (p *Process) Started() bool {
	return p.mon != nil
}

// Running returns true iff process is currently running.
func (p *Process) Running() bool {
	return p.mon != nil && p.mon.running
}

// Exited returns true iff process was running before and is not running currently.
func (p *Process) Exited() bool {
	return p.mon != nil && !p.mon.running
}
