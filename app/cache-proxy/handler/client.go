package handler

import (
	"bufio"
	"net"
)

type Client struct {
	Conn       net.Conn
	IP         string
	Reader     *bufio.Reader
	ServerConn net.Conn
}

func (p *Proxy) NewClient(conn net.Conn) *Client {
	cli := &Client{
		Conn:   conn,
		IP:     conn.RemoteAddr().String(),
		Reader: bufio.NewReader(conn),
		// ServerConn: srvconn,
	}
	return cli
}
