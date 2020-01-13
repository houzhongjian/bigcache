package handler

import (
	"net"

	"github.com/houzhongjian/bigcache/lib/errcode"
	"github.com/houzhongjian/bigcache/lib/packet"
)

type Client struct {
	Conn net.Conn
	IP   string
}

func (master *Master) newClient(conn net.Conn) *Client {
	cli := &Client{
		Conn: conn,
		IP:   conn.RemoteAddr().String(),
	}
	return cli
}

//Write.
func (cli *Client) Write(msg string, num errcode.BigcacheError) {
	buf := packet.NewResponse(msg, num)
	cli.Conn.Write(buf)
}
