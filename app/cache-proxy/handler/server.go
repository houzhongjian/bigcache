package handler

import (
	"log"
	"net"
	"time"

	"github.com/houzhongjian/bigcache/lib/conf"
)

func (p *Proxy) connServer() net.Conn {
	var SrvConn net.Conn
	for {
		conn, err := net.Dial("tcp4", conf.GetString("cache_server"))
		if err != nil {
			log.Printf("err:%+v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}
		SrvConn = conn
		break
	}
	log.Println("server节点连接成功!")
	return SrvConn
}
