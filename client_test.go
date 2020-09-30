//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	rsaPrivate = []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
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
-----END OPENSSH PRIVATE KEY-----`)
	testSrvAddr  = "127.0.0.1:5522"
	testSrvStdin = ""
	testSrvSig   ssh.Signal
)

func init() {
	go sshServer()
	time.Sleep(time.Second * 3)
}

func TestAddClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := GetConfigUserPass("vssh", "vssh")

	vs := New().StartWithContext(ctx)
	vs.AddClient("127.0.0.1:22", config)

	_, ok := vs.clients.get("127.0.0.1:22")
	if !ok {
		t.Error("expect to have 127.0.0.1:22 client but not exist")
	}

	_, ok = vs.clients.get("127.0.0.2:22")
	if ok {
		t.Error("expect to have not 127.0.0.2:22 but it's exist")
	}

	err := vs.AddClient("127.0.0.1:22", nil)
	if err == nil {
		t.Error("client validation failed")
	}

	err = vs.AddClient("127.0.0.1", config)
	if err == nil {
		t.Error("client validation failed")
	}
}

func TestIsSessionsMaxOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vs := New().StartWithContext(ctx)
	vs.AddClient("127.0.0.1:22", &ssh.ClientConfig{}, SetMaxSessions(2))

	client, _ := vs.clients.get("127.0.0.1:22")
	client.curSessions = 2

	if ok := client.isSessionsMaxOut(); !ok {
		t.Error("isSessionMaxOut")
	}
}

func TestSessionsMaxOutRaceQueries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := GetConfigUserPass("vssh", "vssh")

	vs := New().StartWithContext(ctx)
	vs.AddClient(testSrvAddr, config, SetMaxSessions(1), DisableRequestPty())

	vs.Wait(100)

	timeout, _ := time.ParseDuration("5s")
	respChan1 := vs.Run(ctx, "ping", timeout)
	respChan2 := vs.Run(ctx, "ping", timeout)

	resp1 := <-respChan1
	resp2 := <-respChan2

	maxOutCount := 0
	nilErr := 0

	if resp1.err != nil && strings.Contains(resp1.err.Error(), "sessions maxout") {
		maxOutCount++
	}

	if resp1.err == nil {
		nilErr++
	}

	if resp2.err != nil && strings.Contains(resp2.err.Error(), "sessions maxout") {
		maxOutCount++
	}

	if resp2.err == nil {
		nilErr++
	}

	if maxOutCount != 1 || nilErr != 1 {
		t.Error("sessions maxout race failed")
	}

}

func TestTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := GetConfigUserPass("vssh", "vssh")

	vs := New().StartWithContext(ctx)
	vs.AddClient(testSrvAddr, config, SetMaxSessions(1), DisableRequestPty())

	vs.Wait()

	time.Sleep(time.Second)

	timeout, _ := time.ParseDuration("1s")
	respChan := vs.Run(ctx, "ping", timeout)
	now := time.Now()
	resp := <-respChan
	_, _, err := resp.GetText(vs)
	delta := time.Since(now)

	if err == nil || delta.Seconds() > 1.1 {
		t.Error("timeout failed")
	}
}

func TestGoroutineLeak(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := GetConfigUserPass("vssh", "vssh")

	vs := New().StartWithContext(ctx)
	vs.AddClient(testSrvAddr, config, SetMaxSessions(1), DisableRequestPty())

	vs.Wait()

	time.Sleep(time.Second)
	nGoRoutine := runtime.NumGoroutine()

	timeout, _ := time.ParseDuration("6s")
	respChan := vs.Run(ctx, "ping", timeout)
	resp := <-respChan
	resp.GetText(vs)

	time.Sleep(time.Second * 1)
	delta := runtime.NumGoroutine() - nGoRoutine
	if delta > 0 {
		t.Error("goroutine leak issue: expect delta 0 but got, ", delta)
	}
}

func TestQueryWithLabel(t *testing.T) {
	vs := New()
	vs.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout, _ := time.ParseDuration("2s")
	config := GetConfigUserPass("vssh", "vssh")

	labels := map[string]string{"POP": "ORD"}
	vs.AddClient(testSrvAddr, config, SetLabels(labels), DisableRequestPty())

	vs.Wait(100)

	respChan, err := vs.RunWithLabel(ctx, "ping", "POP==LAX", timeout)
	if err != nil {
		t.Fatal(err)
	}

	// should be empty result
	_, ok := <-respChan
	if ok {
		t.Fatal("expect to get false but got", ok)
	}

	respChan, err = vs.RunWithLabel(ctx, "ping", "POP==ORD", timeout)
	if err != nil {
		t.Fatal(err)
	}

	// should be return result
	_, ok = <-respChan
	if !ok {
		t.Fatal("expect to get true but got", ok)
	}

	client, ok := vs.clients.get(testSrvAddr)
	if !ok {
		t.Error("expect test-client but not exist")
	}

	// persistent connection test
	err = client.client.Close()
	if err != nil {
		t.Error("unexpected error as client should be open")
	}

	// wrong query
	_, err = vs.RunWithLabel(ctx, "ping", "POP=ORD", timeout)
	if err == nil {
		t.Fatal(err)
	}

}

func TestOnDemand(t *testing.T) {
	vs := New().Start().OnDemand()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout, _ := time.ParseDuration("2s")
	config := GetConfigUserPass("vssh", "vssh")
	vs.AddClient(testSrvAddr, config, SetMaxSessions(2), DisableRequestPty())

	d, _ := vs.Wait()
	if d != 0 {
		t.Error("expect delay zero but got,", d)
	}

	time.Sleep(time.Second)

	client, ok := vs.clients.get(testSrvAddr)
	if !ok {
		t.Error("expect test-client but not exist")
	}

	if client.client != nil {
		t.Error("expect shouldn't connect in on-demand mode (before query)")
	}

	respChan := vs.Run(ctx, "ping", timeout)
	resp := <-respChan
	if resp.Err() != nil {
		t.Error("unexpect error", resp.Err())
	} else {
		_, _, err := resp.GetText(vs)
		if err == nil {
			t.Error("expect timeout error but err is nil")
		}
	}

	time.Sleep(time.Millisecond * 100)
	if client.client != nil {
		t.Error("expected close client at OnDemand mode")
	}
}

func TestStream(t *testing.T) {
	vs := New().Start()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout, _ := time.ParseDuration("6s")
	config := GetConfigUserPass("vssh", "vssh")
	vs.AddClient(testSrvAddr, config, SetMaxSessions(2), DisableRequestPty())

	vs.Wait()
	respChan := vs.Run(ctx, "ping", timeout)
	resp := <-respChan
	stream := resp.GetStream()
	defer stream.Close()

	counter := 0
	for stream.ScanStdout() {
		stream.BytesStdout()
		counter++
	}

	if counter != 8 {
		t.Error("expect to get 8 lines but got", counter)
	}
}

func sshServer() {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "vssh" && string(pass) == "vssh" {
				return nil, nil
			}
			return nil, fmt.Errorf("password wrong for %q", c.User())
		},
	}

	private, err := ssh.ParsePrivateKey(rsaPrivate)
	if err != nil {
		log.Fatal(err)
	}

	config.AddHostKey(private)

	l, err := net.Listen("tcp", testSrvAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		nConn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(nConn net.Conn) {
			_, chans, reqs, err := ssh.NewServerConn(nConn, config)
			if err != nil {
				log.Fatal(err)
			}

			go ssh.DiscardRequests(reqs)

			for c := range chans {
				handler(c)
			}
		}(nConn)
	}
}

func handler(newChannel ssh.NewChannel) {
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Fatal(err)
	}

	go func(in <-chan *ssh.Request) {
		for req := range in {
			req.Reply(req.Type == "shell", nil)
			if req.Type == "signal" {
				testSrvSig = ssh.Signal(req.Payload[4:])
			}
		}
	}(requests)

	// TestClientAttrRun needs stdin data
	go func() {
		buf := new(bytes.Buffer)
		io.CopyN(buf, channel, 3)
		testSrvStdin = buf.String()
	}()

	for line := range mockPing() {
		io.Copy(channel, bytes.NewBufferString(line+"\n"))
	}

	channel.Close()
}

func mockPing() chan string {
	ch := make(chan string, 4)
	go func() {
		ch <- "PING 8.8.8.8 (8.8.8.8): 56 data bytes"
		for i := 0; i < 4; i++ {
			ch <- fmt.Sprintf("64 bytes from 8.8.8.8: icmp_seq=%d ttl=54 time=2 ms", i)
			time.Sleep(time.Second * 1)
		}
		ch <- "--- 8.8.8.8 ping statistics ---"
		ch <- "4 packets transmitted, 4 packets received, 0.0% packet loss"
		ch <- "round-trip min/avg/max/stddev = 2/2/2/2 ms"
		close(ch)
	}()

	return ch
}

func TestAddClients(t *testing.T) {
	clients := newClients()
	clients.add(&clientAttr{
		addr: "127.0.0.1:22",
	})

	c, ok := clients.get("127.0.0.1:22")
	if !ok {
		t.Error("expect to have a client but nothing")
	}

	if c.addr != "127.0.0.1:22" {
		t.Error("expect to have client 127.0.0.1:22 but got,", c.addr)
	}
}

func TestClientsRace(t *testing.T) {
	var wg sync.WaitGroup
	rand.Seed(time.Now().UnixNano())
	addrs := []string{
		"127.0.0.0:22",
		"127.0.0.1:22",
		"127.0.0.2:22",
		"127.0.0.3:22",
		"127.0.0.4:22",
		"127.0.0.5:22",
		"127.0.0.6:22",
		"127.0.0.7:22",
		"127.0.0.8:22",
		"127.0.0.9:22",
	}

	clients := newClients()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				time.Sleep(time.Nanosecond * time.Duration(rand.Intn(100)))
				clients.add(&clientAttr{addr: addrs[rand.Intn(10)]})
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				time.Sleep(time.Nanosecond * time.Duration(rand.Intn(100)))
				clients.get(addrs[rand.Intn(10)])
			}
		}()
	}

	wg.Wait()
}

func TestNewSession(t *testing.T) {
	vs := New().Start()
	config := GetConfigUserPass("vssh", "vssh")
	vs.AddClient(testSrvAddr, config, SetMaxSessions(2), DisableRequestPty())
	vs.Wait()

	client, _ := vs.clients.get(testSrvAddr)
	session, err := client.newSession()
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	session.Close()

	// enable pty - test server doesn't support
	client.pty.enabled = true
	if _, err = client.newSession(); err == nil {
		t.Fatal("expect error but got", err)
	}
}

func TestClientMemGetShard(t *testing.T) {
	tmp := clientsShardNum
	clients := newClients()
	clientsShardNum = 10
	shard := clients.getShard("10.0.0.1:22")
	clientsShardNum = tmp
	if shard != 9 {
		t.Error("expect shard == 9 but got", shard)
	}
}

func TestClientMem(t *testing.T) {
	addr := "10.0.0.1:22"
	clients := newClients()
	client := &clientAttr{addr: addr}
	clients.add(client)
	c, ok := clients.get(addr)
	if !ok {
		t.Fatal("expect get client returns a client but not exist")
	}

	if c != client {
		t.Error("get client returns wrong client")
	}

	clients.del(addr)
	_, ok = clients.get(addr)
	if ok {
		t.Errorf("expect client %s deleted but still exist", addr)
	}
}

func TestStreamInput(t *testing.T) {
	ch := make(chan []byte, 1)
	r := &Response{
		inChan: ch,
	}

	s := r.GetStream()
	s.Input(bytes.NewBufferString("foo"))
	v := <-ch
	if string(v) != "foo" {
		t.Error("expect foo but got", string(v))
	}
}

func TestStreamScanStdout(t *testing.T) {
	ch := make(chan []byte, 1)
	r := &Response{
		outChan: ch,
	}
	s := r.GetStream()

	s.done = true
	ok := s.ScanStdout()
	if ok {
		t.Error("expect false but got", ok)
	}

	s.done = false
	ch <- []byte("foo")
	ok = s.ScanStdout()
	if !ok {
		t.Fatal("expect true but got", ok)
	}

	if s.TextStdout() != "foo" {
		t.Fatal("expect foo but got", s.stderr)
	}
}

func TestStreamScanStderr(t *testing.T) {
	ch := make(chan []byte, 1)
	r := &Response{
		errChan: ch,
	}
	s := r.GetStream()

	s.done = true
	ok := s.ScanStderr()
	if ok {
		t.Fatal("expect false but got", ok)
	}

	s.done = false
	ch <- []byte("foo")
	ok = s.ScanStderr()
	if !ok {
		t.Fatal("expect true but got", ok)
	}

	if s.TextStderr() != "foo" {
		t.Fatal("expect foo but got", s.stderr)
	}

	if string(s.BytesStderr()) != "foo" {
		t.Fatal("expect foo but got", s.stderr)
	}
}

func TestClientAttrRun(t *testing.T) {
	addr := testSrvAddr
	ch := make(chan *Response, 1)
	client := &clientAttr{
		addr:        addr,
		config:      GetConfigUserPass("vssh", "vssh"),
		maxSessions: 2,
	}

	q := &query{
		respChan:    ch,
		ctx:         context.Background(),
		respTimeout: time.Second * 1,
	}

	// run not connected client
	client.run(q)
	<-ch
	if !errors.Is(client.err, errNotConn) {
		t.Fatalf("expect errNotConn but got %v", client.err)
	}

	client.connect()

	// close conn to make session error
	client.client.Close()
	client.run(q)
	r := <-ch
	if r.err == nil {
		t.Fatal("expect close network connection error")
	}

	client.connect()
	q.cmd = "ping"
	q.respTimeout = time.Second * 6
	go client.run(q)
	r = <-ch
	r.inChan <- []byte("foo")
	time.Sleep(time.Second * 1)
	if testSrvStdin != "foo" {
		t.Fatal("expect stdin foo but got", testSrvStdin)
	}
}

func TestResponseExitStatus(t *testing.T) {
	r := Response{exitStatus: 1}
	if v := r.ExitStatus(); v != 1 {
		t.Error("expect exist-status 1 but got", v)
	}
}

func TestResponseID(t *testing.T) {
	r := Response{id: "127.0.0.1:22"}
	if v := r.ID(); v != "127.0.0.1:22" {
		t.Error("expect id 127.0.0.1:22 but got", v)
	}
}

func TestGetScanners(t *testing.T) {
	vs := New().Start()
	config := GetConfigUserPass("vssh", "vssh")
	vs.AddClient(testSrvAddr, config, SetMaxSessions(2), DisableRequestPty())
	vs.Wait()

	client, _ := vs.clients.get(testSrvAddr)
	session, err := client.newSession()
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = client.getScanners(session, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientConnect(t *testing.T) {
	c := &clientAttr{}
	c.client = &ssh.Client{}
	c.err = nil
	c.connect()

	// it shouldn't connect
	// if c.client != nil && c.err == nil -> refuse connect
	if c.err != nil {
		t.Fatal("souldn't try connect once client connected w/ no error")
	}

	c.client = nil
	c.maxSessions = 0
	c.connect()

	// it shouldn't connect
	// maxSessions zero means out of service
	if c.err != nil {
		t.Fatal("shouldn't try connect when the max sessions is zero")
	}
}

func TestStreamSignal(t *testing.T) {
	vs := New().Start()

	timeout, _ := time.ParseDuration("6s")
	config := GetConfigUserPass("vssh", "vssh")
	vs.AddClient(testSrvAddr, config, SetMaxSessions(2), DisableRequestPty())
	vs.Wait()

	respChan := vs.Run(context.Background(), "ping", timeout)
	resp := <-respChan
	stream := resp.GetStream()
	defer stream.Close()

	stream.Signal(ssh.SIGALRM)
	time.Sleep(time.Millisecond * 100)
	if testSrvSig != ssh.SIGALRM {
		t.Error("expect to get SIGALRM but got", testSrvSig)
	}
}
