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

func TestSetInitNumProcess(t *testing.T) {
	SetInitNumProcess(100)
	if initNumProcess != 100 {
		t.Error("expect initNumProcess 100 but got", initNumProcess)
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
