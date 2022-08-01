package http

import (
	"crypto/tls"
	"io"
	"net/http/httptrace"
	"time"
)

// 记录http response 字节数
type byteCounter struct {
	io.ReadCloser
	n int64
}

func (bc *byteCounter) Read(p []byte) (int, error) {
	n, err := bc.ReadCloser.Read(p)
	bc.n += int64(n)
	return n, err
}

// DNSStart 整个trace链路周期的开始
func (t *transport) DNSStart(_ httptrace.DNSStartInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current.start = time.Now()
}

// DNSDone 域名解析完成
func (t *transport) DNSDone(_ httptrace.DNSDoneInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current.dnsDone = time.Now()
}

func (t *transport) ConnectStart(_, _ string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ts := t.current
	// 连接的是IP而非域名
	if ts.dnsDone.IsZero() {
		ts.start = time.Now()
		ts.dnsDone = ts.start
	}
}

func (t *transport) ConnectDone(net, addr string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.current.connectDone = time.Now()
}

func (t *transport) GotConn(_ httptrace.GotConnInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.current.gotDone = time.Now()
}

func (t *transport) GotFirstResponseByte() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current.responseStart = time.Now()
}

func (t *transport) TLSHandshakeStart() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current.tlsStart = time.Now()
}

func (t *transport) TLSHandshakeDone(_ tls.ConnectionState, _ error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current.tlsDone = time.Now()
}
