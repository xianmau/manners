package manners

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// NewListener wraps an existing listener for use with
// GracefulServer.
//
// Note that you generally don't need to use this directly as
// GracefulServer will automatically wrap any non-graceful listeners
// supplied to it.
func NewListener(l net.Listener) *GracefulListener {
	return &GracefulListener{l, 1}
}

// A gracefulCon wraps a normal net.Conn and tracks the
// last known http state.
type gracefulConn struct {
	net.Conn
	lastHTTPState http.ConnState
}

// A GracefulListener differs from a standard net.Listener in one way: if
// Accept() is called after it is gracefully closed, it returns a
// listenerAlreadyClosed error. The GracefulServer will ignore this
// error.
type GracefulListener struct {
	net.Listener
	open int32
}

// Accept implements the Accept method in the Listener interface.
func (l *GracefulListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		if atomic.LoadInt32(&l.open) == 0 {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}

	gconn := &gracefulConn{conn, 0}
	return gconn, nil
}

// Close tells the wrapped listener to stop listening.  It is idempotent.
func (l *GracefulListener) Close() error {
	if atomic.CompareAndSwapInt32(&l.open, 1, 0) {
		err := l.Listener.Close()
		return err
	}
	return nil
}

type listenerAlreadyClosed struct {
	error
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// direct lift from net/http/server.go
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
