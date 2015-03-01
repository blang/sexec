package sexec

import (
	"syscall"
	"testing"
	"time"
)

var DevNull = devNull{}

type devNull struct{}

func (devNull) Write(p []byte) (int, error) {
	return len(p), nil
}

func TestRun(t *testing.T) {
	p := NewProcess("echo test", DevNull, DevNull)
	err := p.Run()
	if err != nil {
		t.Errorf("Error: %s", err)
	}
}

func TestRunTwice(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	ch := make(chan struct{})
	if err := p.Start(); err != nil {
		t.Errorf("Error: %s", err)
	}
	go func() {
		if err := p.Run(); err != nil {
			if err != ErrorAlreadyRunning {
				t.Errorf("Unexpected error: %s", err)
			}
			close(ch)
		}
	}()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Errorf("Run did not complete in time")
	}
	p.Signal(syscall.SIGTERM)
	p.Wait()
}

func TestSignal(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if err := p.Signal(syscall.SIGHUP); err == nil {
		t.Errorf("Signal should error, process not started")
	}

	// Start
	p.Start()

	stopCh := make(chan struct{})
	go func() {
		p.Wait()
		close(stopCh)
	}()

	if err := p.Signal(syscall.SIGHUP); err != nil {
		t.Errorf("Error on signal: %s", err)
	}
	select {
	case <-time.After(3 * time.Second):
		t.Error("Process was not killed in time")
	case <-stopCh: //success
	}
}

func TestPid(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if _, err := p.Pid(); err == nil {
		t.Errorf("Pid should error")
	}

	// Start
	p.Start()
	if pid, err := p.Pid(); err != nil {
		t.Errorf("Pid error: %s", err)
	} else {
		t.Logf("Pid: %d", pid)
	}
	p.Signal(syscall.SIGTERM)
	p.Wait()

	// Pid after process exists
	if pid, err := p.Pid(); err != nil {
		t.Errorf("Pid error: %s", err)
	} else if pid <= 0 {
		t.Errorf("Malformed Pid: %d", pid)
	}

}

func TestWaitDefault(t *testing.T) {
	p := NewProcess("echo test", DevNull, DevNull)
	if _, err := p.Wait(); err == nil {
		t.Errorf("Expected error")
	}

	p.Start()
	code, err := p.Wait()
	if code != 0 {
		t.Errorf("Wait returned wrong exit code: %d", code)
	}
	if err != nil {
		t.Errorf("Wait returned error: %s", err)
	}
}

func TestWaitTwice(t *testing.T) {
	p := NewProcess("echo test", DevNull, DevNull)
	p.Start()
	code1, err1 := p.Wait()
	if code1 != 0 {
		t.Errorf("Wait returned wrong exit code: %d", code1)
	}
	if err1 != nil {
		t.Errorf("Wait returned error: %s", err1)
	}

	code2, err2 := p.Wait()
	if code2 != 0 {
		t.Errorf("Wait returned wrong exit code: %d", code2)
	}
	if err2 != nil {
		t.Errorf("Wait returned error: %s", err2)
	}
}

func TestWaitCustomExitCode(t *testing.T) {
	p := NewProcess("exit 113", DevNull, DevNull)
	p.Start()
	if code, _ := p.Wait(); code != 113 {
		t.Errorf("Wait returned wrong exit code: %d", code)
	}
}

func TestWaitSignalSIGHUP(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	p.Start()

	if err := p.Signal(syscall.SIGHUP); err != nil {
		t.Logf("Error on signal: %s", err)
	}

	if code, err := p.Wait(); code != 129 {
		t.Errorf("Wait returned wrong exit code: %d error %s\n", code, err)
	}
}

func TestWaitSignalSIGTERM(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	p.Start()

	if err := p.Signal(syscall.SIGTERM); err != nil {
		t.Logf("Error on signal: %s", err)
	}

	if code, err := p.Wait(); code != 128+15 {
		t.Errorf("Wait returned wrong exit code: %d error %s\n", code, err)
	}
}

func TestWaitCh(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if ch := p.WaitCh(); ch != nil {
		t.Errorf("Expected nil channel")
	}
	p.Start()
	ch := p.WaitCh()
	select {
	case <-ch:
		t.Errorf("Unexpected channel read") // not killed yet
	default:
	}

	p.Signal(syscall.SIGTERM)

	<-ch

	select {
	case <-time.After(3 * time.Second):
		t.Error("Process was not killed in time")
	case <-ch: //success
	}
}

func TestWaitOnCh(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if err := p.WaitOnCh(make(chan struct{})); err == nil {
		t.Errorf("Expected error")
	}
	p.Start()
	ch := make(chan struct{})
	if err := p.WaitOnCh(ch); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	select {
	case <-ch:
		t.Errorf("Unexpected channel read")
	default:
	}

	p.Signal(syscall.SIGTERM)

	select {
	case <-time.After(3 * time.Second):
		t.Error("Process was not killed in time")
	case <-ch: //success
	}
}

func TestWaitOnNilCh(t *testing.T) {
	p := NewProcess("echo test", DevNull, DevNull)
	p.Start()
	defer func() {
		if recover() == nil {
			t.Errorf("Should have panicked")
		}
	}()
	p.WaitOnCh(nil)
}

func TestExitCode(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if code, err := p.ExitCode(); err == nil {
		t.Errorf("Unexpected exit code: %d", code)
	}

	p.Start()
	if _, err := p.ExitCode(); err == nil {
		t.Errorf("Expected error")
	}

	p.Signal(syscall.SIGTERM)

	<-p.WaitCh()

	if code, err := p.ExitCode(); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if code != 128+15 {
		t.Errorf("Wait returned wrong exit code: %d error %s\n", code, err)
	}
}

func TestExitCode0(t *testing.T) {
	p := NewProcess("echo 1", DevNull, DevNull)
	if code, err := p.ExitCode(); err == nil {
		t.Errorf("Unexpected exit code: %d", code)
	}

	p.Run()

	if code, err := p.ExitCode(); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if code != 0 {
		t.Errorf("Wait returned wrong exit code: %d error %s\n", code, err)
	}
}

func TestSuccess(t *testing.T) {
	p := NewProcess("echo test", DevNull, DevNull)
	if p.Success() {
		t.Errorf("Expected: No success")
	}
	p.Start()
	p.Wait()

	if !p.Success() {
		t.Errorf("No success")
	}
}

func TestSuccessFail(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	p.Start()
	p.Signal(syscall.SIGTERM)
	p.Wait()

	if p.Success() {
		t.Errorf("Success, but was killed")
	}
}

func TestStarted(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if p.Started() {
		t.Errorf("Process should not be started")
	}
	p.Start()
	if !p.Started() {
		t.Errorf("Process should be started")
	}
	p.Signal(syscall.SIGTERM)
	p.Wait()
	if !p.Started() {
		t.Errorf("Process should be started")
	}
}

func TestExited(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if p.Exited() {
		t.Errorf("Process not started, but not exited")
	}

	p.Start()
	if p.Exited() {
		t.Errorf("Process should be running")
	}

	p.Signal(syscall.SIGTERM)
	p.Wait()
	if !p.Exited() {
		t.Errorf("Process should not be running after SIGTERM")
	}
}

func TestRunning(t *testing.T) {
	p := NewProcess("sleep 10", DevNull, DevNull)
	if p.Running() {
		t.Errorf("Process should not be running")
	}

	p.Start()
	if !p.Running() {
		t.Errorf("Process should be running")
	}

	p.Signal(syscall.SIGTERM)
	p.Wait()
	if p.Running() {
		t.Errorf("Process should not be running after SIGTERM")
	}
}
