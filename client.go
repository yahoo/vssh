//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	errMaxSessions = errors.New("sessions maxout")
	errUnreachable = errors.New("client unreachable")
	errTimeout     = errors.New("execution timeout")
	errSessNotEst  = errors.New("session not established")
	errNotConn     = errors.New("client hasn't connected")

	maxOutChanBuf = 100
	maxErrChanBuf = 100
	maxInChanBuf  = 100

	dialTimeoutSec = 5
)

// TimeoutError represents timeout error.
type TimeoutError struct {
	error
}

// MaxSessionsError represents max sessions error.
type MaxSessionsError struct {
	error
}

type clientStats struct {
	errCounter uint64
	errRecent  uint64
}

// clientAttr represents client attributes
type clientAttr struct {
	addr        string
	labels      map[string]string
	config      *ssh.ClientConfig
	client      *ssh.Client
	logger      *log.Logger
	maxSessions uint8
	curSessions uint8
	lastUpdate  time.Time
	pty         pty
	stats       clientStats
	err         error

	sync.RWMutex
}

// Response represents the response for given session.
type Response struct {
	id string

	outChan  chan []byte
	inChan   chan []byte
	errChan  chan []byte
	sigChan  chan ssh.Signal
	toCancel chan struct{}

	session    *ssh.Session
	exitStatus int
	err        error
}

// Stream represents data stream for given response.
// It provides convenient interfaces to get the returned
// data real-time.
type Stream struct {
	r      *Response
	stdout []byte
	stderr []byte
	done   bool
}

type connect struct {
	*clientAttr
}

// pty represents pty attribute
type pty struct {
	enabled bool
	term    string
	modes   ssh.TerminalModes
	wide    uint
	height  uint
}

// run executes the command on the client
func (c *clientAttr) run(q *query) {
	var (
		wg sync.WaitGroup

		done = make(chan struct{})

		rcOut = make(chan []byte, maxOutChanBuf)
		rcIn  = make(chan []byte, maxInChanBuf)
		rcErr = make(chan []byte, maxErrChanBuf)
		rcSig = make(chan ssh.Signal, 1)
	)

	if c.client == nil {
		setErr(c, errNotConn)
		q.errResp(c.addr, errNotConn)
		return
	}

	if c.isSessionsMaxOut() {
		q.errResp(c.addr, MaxSessionsError{errMaxSessions})
		return
	}

	c.incSessions()

	session, err := c.newSession()
	if err != nil {
		setErr(c, err)
		q.errResp(c.addr, err)
		return
	}

	writer, err := session.StdinPipe()
	if err != nil {
		setErr(c, err)
		q.errResp(c.addr, err)
		return
	}

	scanOut, scanErr, err := c.getScanners(session, q.limitReadOut, q.limitReadErr)
	if err != nil {
		setErr(c, err)
		q.errResp(c.addr, err)
		return
	}

	resp := &Response{
		id: c.addr,

		outChan: rcOut,
		inChan:  rcIn,
		errChan: rcErr,
		sigChan: rcSig,

		session: session,
	}

	resp.setTimeout(q.respTimeout)
	q.respChan <- resp

	session.Start(q.cmd)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for scanOut.Scan() {
			select {
			case rcOut <- scanOut.Bytes():
			default:
				c.logger.Println("msg stdout has been dropped")
			}
		}

		if err := scanOut.Err(); err != nil {
			c.logger.Println(err)
		}

		close(done)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for scanErr.Scan() {
			select {
			case rcErr <- scanErr.Bytes():
			default:
				c.logger.Println("msg stderr has been dropped")
			}
		}

		if err := scanErr.Err(); err != nil {
			c.logger.Println(err)
		}
	}()

LOOP:
	for {
		select {
		case in := <-rcIn:
			fmt.Fprint(writer, string(in))
		case sig := <-rcSig:
			session.Signal(sig)
		case <-q.ctx.Done():
			session.Close()
		case <-done:
			break LOOP
		}

	}

	if err = session.Wait(); err != nil {
		switch e := err.(type) {
		case *ssh.ExitError:
			resp.exitStatus = e.ExitStatus()
		}
	}

	session.Close()
	c.decSessions()

	wg.Wait()

	close(rcOut)
	close(rcErr)
}

func (c *clientAttr) newSession() (*ssh.Session, error) {
	if c.client == nil {
		c.decSessions()
		return nil, errUnreachable
	}

	cancelTimeout := make(chan struct{})

	go func() {
		select {
		case <-time.After(time.Second * 5):
			c.client.Close()
		case <-cancelTimeout:
		}
	}()

	s, err := c.client.NewSession()
	close(cancelTimeout)

	if err != nil {
		c.decSessions()
		return s, err
	}

	if c.pty.enabled {
		err := s.RequestPty(
			c.pty.term,
			int(c.pty.height),
			int(c.pty.wide),
			c.pty.modes)

		if err != nil {
			c.decSessions()
			return nil, err
		}
	}

	return s, err
}

func (c *clientAttr) isSessionsMaxOut() bool {
	return c.getSessions() >= c.maxSessions
}

func (c *clientAttr) getScanners(s *ssh.Session, lOut, lErr int64) (*bufio.Scanner, *bufio.Scanner, error) {
	var (
		scanOut *bufio.Scanner
		scanErr *bufio.Scanner
		err     error
	)

	readerOut, err := s.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	readerErr, err := s.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if lOut > 0 {
		lReaderOut := io.LimitReader(readerOut, lOut)
		scanOut = bufio.NewScanner(lReaderOut)
	} else {
		scanOut = bufio.NewScanner(readerOut)
	}

	if lErr > 0 {
		lReaderErr := io.LimitReader(readerErr, lErr)
		scanErr = bufio.NewScanner(lReaderErr)
	} else {
		scanErr = bufio.NewScanner(readerErr)
	}

	return scanOut, scanErr, nil
}

func (c *clientAttr) setErr(err error) {
	c.stats.errRecent++
	c.stats.errCounter++
	c.lastUpdate = time.Now()
	c.err = err
}

func (c *clientAttr) getErr() error {
	c.RLock()
	defer c.RUnlock()
	return c.err
}

func (c *clientAttr) getClient() *ssh.Client {
	c.RLock()
	defer c.RUnlock()
	return c.client
}

func (c *clientAttr) labelMatch(v *visitor) bool {
	if len(c.labels) < 1 {
		return false
	}

	ok, err := exprEval(v, c.labels)
	if err != nil {
		return false
	}

	return ok
}

func (c *clientAttr) connect() {
	c.Lock()
	defer c.Unlock()

	// already connected w/o error
	if c.client != nil && c.err == nil {
		return
	}

	// out of service
	if c.maxSessions == 0 {
		return
	}

	timeout := time.Duration(dialTimeoutSec) * time.Second
	conn, err := net.DialTimeout("tcp", c.addr, timeout)
	if err != nil {
		c.setErr(err)
		return
	}

	sshConn, chans, req, err := ssh.NewClientConn(conn, c.addr, c.config)
	if err != nil {
		conn.Close()
		c.setErr(err)
		return
	}

	c.client = ssh.NewClient(sshConn, chans, req)
	c.lastUpdate = time.Now()
	c.err = nil
	c.stats.errRecent = 0
}

func (c *clientAttr) close() {
	c.Lock()
	defer c.Unlock()
	if c.curSessions == 0 {
		c.client.Close()
		c.client = nil
	}
}

func (c *clientAttr) incSessions() {
	c.Lock()
	defer c.Unlock()
	c.curSessions++
}

func (c *clientAttr) decSessions() {
	c.Lock()
	defer c.Unlock()
	c.curSessions--
}

func (c *clientAttr) getSessions() uint8 {
	c.Lock()
	defer c.Unlock()
	return c.curSessions
}

func (c *connect) run(v *VSSH) {
	c.connect()
}

// SetTimeout sets timeout for the given response
func (r *Response) setTimeout(t time.Duration) {
	r.toCancel = make(chan struct{})

	go func() {
		select {
		case <-r.toCancel:
			return
		case <-time.After(t):
			r.err = TimeoutError{errTimeout}
			r.session.Close()
		}
	}()
}

func (r *Response) cancelTimeout() {
	close(r.toCancel)
}

// GetText gets the final result of the given response.
func (r *Response) GetText(v *VSSH) (string, string, error) {
	var (
		data    []byte
		outDone bool
		errDone bool
		ok      bool
	)

	stream := r.GetStream()
	defer stream.Close()

	bufOut := v.bufPool.Get().(*bytes.Buffer)
	bufErr := v.bufPool.Get().(*bytes.Buffer)

	defer v.bufPool.Put(bufOut)
	defer v.bufPool.Put(bufErr)

	bufOut.Reset()
	bufErr.Reset()

	for {
		select {
		case data, ok = <-stream.r.outChan:
			if ok {
				bufOut.Write(append(data, '\n'))
			} else {
				outDone = true
			}
		case data, ok = <-stream.r.errChan:
			if ok {
				bufErr.Write(append(data, '\n'))
			} else {
				errDone = true
			}
		}

		if outDone && errDone {
			break
		}
	}

	return bufOut.String(), bufErr.String(), stream.Err()
}

// Err returns response error.
func (r *Response) Err() error {
	return r.err
}

// ID returns response identification.
func (r *Response) ID() string {
	return r.id
}

// GetStream constructs a new stream from a response.
func (r *Response) GetStream() *Stream {
	if r.err != nil {
		return nil
	}

	return &Stream{
		r: r,
	}
}

// ExitStatus returns the exit status of the remote command.
func (r *Response) ExitStatus() int {
	return r.exitStatus
}

// ScanStdout provides a convenient interface for reading stdout
// which it connected to remote host. It reads a line and buffers
// it. The TextStdout() or BytesStdout() methods return the buffer
// in string or bytes.
func (s *Stream) ScanStdout() bool {
	if s.done {
		return false
	}

	var ok bool

	s.stdout, ok = <-s.r.outChan

	return ok
}

// TextStdout returns the most recent data scanned by ScanStdout as string.
func (s *Stream) TextStdout() string {
	return string(s.stdout)
}

// BytesStdout returns the most recent data scanned by ScanStdout as bytes.
func (s *Stream) BytesStdout() []byte {
	return s.stdout
}

// ScanStderr provides a convenient interface for reading stderr
// which it connected to remote host. It reads a line and buffers
// it. The TextStdout() or BytesStdout() methods return the buffer
// in string or bytes.
func (s *Stream) ScanStderr() bool {
	if s.done {
		return false
	}

	var ok bool

	s.stderr, ok = <-s.r.errChan

	return ok
}

// TextStderr returns the most recent data scanned by ScanStderr as string.
func (s *Stream) TextStderr() string {
	return string(s.stderr)
}

// BytesStderr returns the most recent data scanned by ScanStderr as bytes.
func (s *Stream) BytesStderr() []byte {
	return s.stderr
}

// Close cleans up the stream's response.
func (s *Stream) Close() error {
	if s.r == nil || s.r.session == nil {
		return errSessNotEst
	}

	s.done = true

	s.r.session.Close()
	s.r.cancelTimeout()

	return nil
}

// Err returns stream response error.
func (s *Stream) Err() error {
	return s.r.err
}

// Signal sends the given signal to remote process.
func (s *Stream) Signal(sig ssh.Signal) {
	s.r.sigChan <- sig
}

// Input writes the given reader to remote command's standard
// input when the command starts.
func (s *Stream) Input(in io.Reader) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(in)
	s.r.inChan <- buf.Bytes()
}

// setErr is a helper func to update error with mutex
func setErr(c *clientAttr, err error) {
	c.Lock()
	defer c.Unlock()
	c.setErr(err)
}
