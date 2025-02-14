package protocol

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"

	"github.com/nsqio/nsq/internal/lg"
)

type TCPHandler interface {
	Handle(net.Conn)
}

func TCPServer(listener net.Listener, handler TCPHandler, logf lg.AppLogFunc) error {
	logf(lg.INFO, "TCP: listening on %s", listener.Addr())

	var wg sync.WaitGroup

	for {
		// 等待客户端的连接， 会阻塞
		clientConn, err := listener.Accept()
		if err != nil {
			// net.Error.Temporary() is deprecated, but is valid for accept
			// this is a hack to avoid a staticcheck error
			if te, ok := err.(interface{ Temporary() bool }); ok && te.Temporary() {
				logf(lg.WARN, "temporary Accept() failure - %s", err)
				runtime.Gosched()
				continue
			}
			// theres no direct way to detect this error because it is not exposed
			if !errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("listener.Accept() error - %s", err)
			}
			break
		}

		// 连接已建立
		wg.Add(1)
		// 开启一个goroutine 处理这个连接的数据I/O
		go func() {
			handler.Handle(clientConn)
			wg.Done()
		}()
	}

	// wait to return until all handler goroutines complete
	wg.Wait()

	logf(lg.INFO, "TCP: closing %s", listener.Addr())

	return nil
}
