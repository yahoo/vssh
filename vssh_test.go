package vssh

import (
	"log"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestForceReConn(t *testing.T) {
	vs := New()
	cfg := GetConfigUserPass("vssh", "vssh")
	err := vs.AddClient("127.0.0.1:22", cfg)
	if err != nil {
		t.Error(err)
	}

	vs.ForceReConn("127.0.0.1:22")
	if len(vs.actionQ) != 2 {
		t.Fatal("expect to have two tasks but got", len(vs.actionQ))
	}

	if err := vs.ForceReConn("127.0.0.2:22"); err == nil {
		t.Fatal("expect to have not exist error")
	}
}

func TestIncreaseProc(t *testing.T) {
	vs := New().Start()
	time.Sleep(time.Millisecond * 500)
	p1 := vs.stats.processes
	vs.IncreaseProc(100)
	time.Sleep(time.Millisecond * 500)
	p2 := vs.stats.processes

	if p2-p1 != 100 {
		t.Error("unexpect increase proc")
	}
}

func TestDecreaseProc(t *testing.T) {
	vs := New().Start()
	time.Sleep(time.Millisecond * 500)
	p1 := vs.stats.processes
	vs.DecreaseProc(100)
	time.Sleep(time.Millisecond * 500)
	p2 := vs.stats.processes

	if p1-p2 != 100 {
		t.Error("unexpect decrease proc")
	}

}

func TestCurrentProc(t *testing.T) {
	vs := New()
	vs.SetInitNumProc(100)
	vs.Start()
	time.Sleep(time.Millisecond * 500)
	n := vs.CurrentProc()
	if n != 100 {
		t.Error("expect 100 current processes but got", n)
	}

}

func TestSetInitNumProc(t *testing.T) {
	vs := New()
	vs.SetInitNumProc(100)
	if initNumProc != 100 {
		t.Error("expect initNumProcess 100 but got", initNumProc)
	}
}

func TestSetClientsShardNumber(t *testing.T) {
	SetClientsShardNumber(20)
	if clientsShardNum != 20 {
		t.Error("expect clientsShardNum 20 but got", clientsShardNum)
	}
}

func TestSetLogger(t *testing.T) {
	vs := New()
	logger := &log.Logger{}
	vs.SetLogger(logger)
	if vs.logger != logger {
		t.Error("setlogger failed")
	}
}

func TestRequestPty(t *testing.T) {
	f := RequestPty("xterm", 40, 80, ssh.TerminalModes{ssh.ECHO: 0})
	c := clientAttr{}
	f(&c)

	if !c.pty.enabled {
		t.Error("expect pty enabled but disabled")
	}

	if c.pty.term != "xterm" {
		t.Error("expect terminal xterm but got", c.pty.term)
	}

	if c.pty.height != 40 {
		t.Error("expect to have height 40 but got", c.pty.height)
	}

	if c.pty.wide != 80 {
		t.Error("expect to have wide 80 but got", c.pty.wide)
	}
}

func TestSetLimitReaderStdout(t *testing.T) {
	f := SetLimitReaderStdout(1024)
	q := &query{}
	f(q)
	if q.limitReadOut != 1024 {
		t.Error("expect limitReadOut 1024 but got", q.limitReadOut)
	}
}

func TestSetLimitReaderStderr(t *testing.T) {
	f := SetLimitReaderStderr(1024)
	q := &query{}
	f(q)
	if q.limitReadErr != 1024 {
		t.Error("expect limitReadErr 1024 but got", q.limitReadErr)
	}
}

func TestGetConfigPEM(t *testing.T) {
	_, err := GetConfigPEM("vssh", "notexitfile")
	if err == nil {
		t.Error("expect error but nil")
	}

	_, err = GetConfigPEM("vssh", "vssh.go")
	if err == nil {
		t.Error("expect error but nil")
	}
}
