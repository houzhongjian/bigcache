package handler

import "net"

type Client struct {
	Conn net.Conn
	IP   string
}

func (c *Cache) NewClient(conn net.Conn) *Client {
	cli := &Client{
		Conn: conn,
		IP:   conn.RemoteAddr().String(),
	}
	return cli
}
