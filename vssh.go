//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

// Package vssh is a Go library to handle tens of thousands SSH connections and execute
// the command with higher-level API for building network device / server automation.
//
//	run(ctx, command, timeout)
//	runWithLabel(ctx, command, timeout, "OS == Ubuntu && POP == LAX")
//
// By calling the run method vssh sends the given command to all available clients or
// based on your query it runs the command on the specific clients and the results of
// the ran command can be received in two options, streaming or final result.In streaming
// you can get line by line from command’s stdout / stderr in real time or in case of
// non-real time you can get the whole of the lines together.
package vssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	defaultMaxSessions  uint8  = 3
	maxErrRecent        uint64 = 10
	maxEstablishedRetry        = 20
	actionQueueSize            = 1000
	initNumProc                = 1000
	resetErrRecentDur          = time.Duration(300) * time.Second
	reConnDur                  = time.Duration(10) * time.Second

	errSSHConfig = errors.New("ssh config can not be nil")
	errNotExist  = errors.New("not exist")
)

// VSSH represents VSSH instance.
type VSSH struct {
	clients clients
	logger  *log.Logger
	stats   stats
	mode    bool
	bufPool sync.Pool

	actionQ chan task
	procSig chan struct{}
	procCtl chan struct{}
}

type stats struct {
	queries   uint64
	connects  uint64
	processes uint64
}

type task interface {
	run(v *VSSH)
}

// ClientOption represents client optional parameters.
type ClientOption func(c *clientAttr)

// RunOption represents run optional parameters.
type RunOption func(q *query)

// New constructs a new VSSH instance.
func New() *VSSH {
	return &VSSH{
		clients: newClients(),
		actionQ: make(chan task, actionQueueSize),
		logger:  log.New(os.Stdout, "vssh: ", log.Lshortfile),

		procSig: make(chan struct{}, 1),
		procCtl: make(chan struct{}, 1),

		bufPool: sync.Pool{
			New: func() interface{} { return new(bytes.Buffer) },
		},

		mode: false,
	}
}

// OnDemand changes VSSH connection behavior. By default VSSH
// connects to all of the clients before any run request and
// it maintains the authenticated SSH connection to all clients.
// We can call this "persistent SSH connection" but with
// OnDemand it tries to connect to clients once the run requested
// and it closes the appropriate connection once the response data returned.
func (v *VSSH) OnDemand() *VSSH {
	v.mode = true
	return v
}

// AddClient adds a new SSH client to VSSH.
func (v *VSSH) AddClient(addr string, config *ssh.ClientConfig, opts ...ClientOption) error {
	client := &clientAttr{
		addr:        addr,
		config:      config,
		maxSessions: defaultMaxSessions,
		logger:      v.logger,
		pty: pty{
			enabled: true,
			term:    "xterm",
			modes: ssh.TerminalModes{
				ssh.ECHO:          0,
				ssh.TTY_OP_ISPEED: 14400,
				ssh.TTY_OP_OSPEED: 14400,
			},
			wide:   80,
			height: 40,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	if err := clientValidation(client); err != nil {
		return err
	}

	v.clients.add(client)

	if !v.mode {
		v.actionQ <- &connect{client}
	}

	return nil
}

// SetMaxSessions sets maximum sessions for given client.
func SetMaxSessions(n int) ClientOption {
	return func(c *clientAttr) {
		c.maxSessions = uint8(n)
	}
}

// RequestPty sets the pty parameters.
func RequestPty(term string, h, w uint, modes ssh.TerminalModes) ClientOption {
	return func(c *clientAttr) {
		c.pty = pty{
			enabled: true,
			term:    term,
			modes:   modes,
			wide:    w,
			height:  h,
		}
	}
}

// DisableRequestPty disables the pty.
func DisableRequestPty() ClientOption {
	return func(c *clientAttr) {
		c.pty.enabled = false
	}
}

// SetLabels sets labels for a client.
func SetLabels(labels map[string]string) ClientOption {
	return func(c *clientAttr) {
		c.labels = labels
	}
}

func clientValidation(c *clientAttr) error {
	if c.config == nil {
		return errSSHConfig
	}

	_, _, err := net.SplitHostPort(c.addr)
	if err != nil {
		return err
	}

	return nil
}

// Start starts vSSH, including action queue and re-connect procedures.
// You can construct and start the vssh like below:
//	vs := vssh.New().Start()
func (v *VSSH) Start() *VSSH {
	ctx := context.Background()
	go v.process(ctx)
	go v.reConnect(ctx)

	for i := 0; i < initNumProc; i++ {
		v.procCtl <- struct{}{}
	}

	return v
}

// StartWithContext is same as Run but it accepts external context.
func (v *VSSH) StartWithContext(ctx context.Context) *VSSH {
	go v.process(ctx)
	go v.reConnect(ctx)

	for i := 0; i < initNumProc; i++ {
		v.procCtl <- struct{}{}
	}

	return v
}

func (v *VSSH) process(ctx context.Context) {
	for {
		go func() {
			for {
				select {
				case a := <-v.actionQ:
					switch b := a.(type) {
					case *connect:
						atomic.AddUint64(&v.stats.connects, 1)
						b.run(v)
					case *query:
						atomic.AddUint64(&v.stats.queries, 1)
						b.run(v)
					}
				case <-v.procSig:
					atomic.AddUint64(&v.stats.processes, ^uint64(0))
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		<-v.procCtl
		v.stats.processes++
	}
}

// IncreaseProc adds more processes / workers.
func (v *VSSH) IncreaseProc(n ...int) {
	num := 1
	if len(n) > 0 {
		num = n[0]
	}

	for i := 0; i < num; i++ {
		v.procCtl <- struct{}{}
	}
}

// DecreaseProc destroys the idle processes / workers.
func (v *VSSH) DecreaseProc(n ...int) {
	num := 1
	if len(n) > 0 {
		num = n[0]
	}

	for i := 0; i < num; i++ {
		v.procSig <- struct{}{}
	}
}

// CurrentProc returns number of running processes / workers.
func (v *VSSH) CurrentProc() uint64 {
	return v.stats.processes
}

// SetInitNumProc sets the initial number of processes / workers.
//
// You need to set this number right after creating vssh.
//	vs := vssh.New()
//	vs.SetInitNumProc(200)
//	vs.Start()
// There are two other methods in case you need to change
// the settings in the middle of your code.
//	IncreaseProc(n int)
//	DecreaseProc(n int)
func (v *VSSH) SetInitNumProc(n int) {
	initNumProc = n
}

// Run sends a new run query with given context, command and timeout.
//
// timeout allows you to set a limit on the length of time the command
// will run for. You can cancel the running command by context.WithCancel.
func (v *VSSH) Run(ctx context.Context, cmd string, timeout time.Duration, opts ...RunOption) chan *Response {
	respChan := make(chan *Response, 100)

	q := &query{
		ctx:         ctx,
		cmd:         cmd,
		respChan:    respChan,
		respTimeout: timeout,
	}

	for _, opt := range opts {
		opt(q)
	}

	v.actionQ <- q

	return respChan
}

// RunWithLabel runs the command on the specific clients which
// they matched with given query statement.
//	labels := map[string]string {
//  	"POP" : "LAX",
//  	"OS" : "JUNOS",
//	}
//	// sets labels to a client
//	vs.AddClient(addr, config, vssh.SetLabels(labels))
//	// run the command with label
//	vs.RunWithLabel(ctx, cmd, timeout, "POP == LAX || POP == DCA) && OS == JUNOS")
func (v *VSSH) RunWithLabel(ctx context.Context, cmd, queryStmt string, timeout time.Duration, opts ...RunOption) (chan *Response, error) {
	vis, err := parseExpr(queryStmt)
	if err != nil {
		return nil, err
	}

	respChan := make(chan *Response, 100)

	q := &query{
		ctx:           ctx,
		cmd:           cmd,
		stmt:          queryStmt,
		compiledQuery: vis,
		respChan:      respChan,
		respTimeout:   timeout,
	}

	for _, opt := range opts {
		opt(q)
	}

	v.actionQ <- q

	return respChan, nil
}

// SetLimitReaderStdout sets limit for stdout reader.
//	respChan := vs.Run(ctx, cmd, timeout, vssh.SetLimitReaderStdout(1024))
func SetLimitReaderStdout(n int64) RunOption {
	return func(q *query) {
		q.limitReadOut = n
	}
}

// SetLimitReaderStderr sets limit for stderr reader.
func SetLimitReaderStderr(n int64) RunOption {
	return func(q *query) {
		q.limitReadErr = n
	}
}

func (v *VSSH) reConnect(ctx context.Context) {
	if v.mode {
		return
	}
	ticker := time.NewTicker(reConnDur)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for client := range v.clients.enum() {
				client.Lock()
				if client.err != nil && client.stats.errRecent < maxErrRecent {
					if client.client != nil {
						client.client.Close()
					}
					v.actionQ <- &connect{client}
				} else if time.Since(client.lastUpdate) > resetErrRecentDur {
					client.stats.errRecent = 0
				}
				client.Unlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

// ForceReConn reconnects the client immediately.
func (v *VSSH) ForceReConn(addr string) error {
	client, ok := v.clients.get(addr)
	if !ok {
		return errNotExist
	}

	if client.client != nil {
		client.client.Close()
	}

	v.actionQ <- &connect{client}

	return nil
}

// Wait stands by until percentage of the clients have been processed.
// An optional percentage can be passed as an argument - otherwise the default
// value of 100% is used.
func (v *VSSH) Wait(p ...int) (float64, error) {
	var (
		start = time.Now()
		retry = 0
		pct   = 100
	)

	if v.mode {
		return 0, nil
	}

	if len(p) > 0 {
		pct = p[0]
	}

	for {
		total := 0
		established := 0

		retry++
		for client := range v.clients.enum() {
			total++

			if client.getClient() != nil {
				established++
			}
		}

		time.Sleep(time.Millisecond * 500)

		if total == 0 || (established*100/total) >= pct {
			break
		}
		if retry > maxEstablishedRetry {
			return time.Since(start).Seconds(), fmt.Errorf("wait established timeout")
		}
	}

	return time.Since(start).Seconds(), nil
}

// SetLogger sets external logger.
func (v *VSSH) SetLogger(l *log.Logger) {
	v.logger = l
}

// SetClientsShardNumber sets the clients shard number.
//
// vSSH uses map data structure to keep the clients
// data in the memory. Sharding helps to have better performance
// on write/read with mutex. This setting can be tuned if needed.
func SetClientsShardNumber(n int) {
	clientsShardNum = n
}
