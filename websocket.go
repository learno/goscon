package main

import (
	"io"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/xjdrew/glog"
)

type websocketConn struct {
	*net.TCPConn
	readTimeout time.Duration

	upgraded bool
	length   int64
	offset   int64
	mask     [4]byte
}

func readMaskData(conn *websocketConn, buf []byte, remain int64) (n int, err error) {
	sz := int64(len(buf))
	if sz > remain {
		sz = remain
	}

	b := buf[:sz]
	n, err = io.ReadFull(conn.TCPConn, b)
	if err != nil {
		return
	}

	ws.Cipher(b, conn.mask, int(conn.offset))
	conn.offset += sz
	return
}

func (conn *websocketConn) Read(b []byte) (int, error) {
	if conn.readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(conn.readTimeout))
	}

	if !conn.upgraded {
		_, err := ws.Upgrade(conn.TCPConn)
		if err != nil {
			return 0, err
		}
		conn.upgraded = true

		if glog.V(1) {
			glog.Infof("upgrade websocket connection: addr=%s", conn.RemoteAddr())
		}
	}

	remain := conn.length - conn.offset
	if remain > 0 {
		return readMaskData(conn, b, remain)
	}

	for {
		header, err := ws.ReadHeader(conn.TCPConn)
		if err != nil {
			return 0, err
		}

		switch header.OpCode {
		case ws.OpClose:
			return 0, io.EOF
		case ws.OpPing:
			payload := make([]byte, header.Length)
			if _, err := io.ReadFull(conn.TCPConn, payload); err != nil {
				return 0, err
			}
			if header.Masked {
				ws.Cipher(payload, header.Mask, 0)
			}
			if err := ws.WriteFrame(conn.TCPConn, ws.NewPongFrame(payload)); err != nil {
				return 0, err
			}
			continue
		default:
			conn.length = header.Length
			conn.offset = 0
			conn.mask = header.Mask
			return readMaskData(conn, b, header.Length)
		}
	}
}

func (conn *websocketConn) Write(b []byte) (int, error) {
	f := ws.NewBinaryFrame(b)
	err := ws.WriteFrame(conn.TCPConn, f)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// WebsocketListener .
type WebsocketListener struct {
	net.Listener
}

// Accept .
func (l *WebsocketListener) Accept() (conn net.Conn, err error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return
	}

	keepalive := configItemBool("websocket_option.keepalive")
	keepaliveInterval := configItemTime("websocket_option.keepalive_interval")
	keepaliveCount := configItemInt("websocket_option.keepalive_count")
	readTimeout := configItemTime("websocket_option.read_timeout")

	t := c.(*net.TCPConn)
	t.SetKeepAliveConfig(net.KeepAliveConfig{
		Enable:   keepalive,
		Idle:     keepaliveInterval,
		Interval: keepaliveInterval,
		Count:    keepaliveCount,
	})
	// t.SetLinger(0)

	if glog.V(1) {
		glog.Infof("accept new websocket connection: addr=%s", c.RemoteAddr())
	}

	conn = &websocketConn{TCPConn: t, readTimeout: readTimeout}
	return
}

// NewWebsocketListener creates a new WebsocketListener
func NewWebsocketListener(laddr string) (*WebsocketListener, error) {
	ln, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}
	return &WebsocketListener{ln}, nil
}
