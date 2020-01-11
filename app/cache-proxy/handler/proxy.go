package handler

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/houzhongjian/bigcache/lib/conf"
)

type Proxy struct {
	Addr string
	Ch   chan bool
}

//NewProxy.
func NewProxy() Proxy {
	p := Proxy{
		Addr: fmt.Sprintf(":%s", conf.GetString("port")),
		Ch:   make(chan bool),
	}
	return p
}

func (p *Proxy) Start() {
	p.start()
}

func (p *Proxy) start() {
	//监听tcp端口.
	go p.checkProxyStart()
	p.listen()
}

//checkProxyStart 检查是否启动成功.
func (p *Proxy) checkProxyStart() {
	for {
		select {
		case msg := <-p.Ch:
			if msg {
				p.welcome()
			} else {
				log.Println("启动失败")
			}
		}
	}
}

//welcome 命令行界面.
func (p *Proxy) welcome() {
	log.Println("Bigcache Proxy")
}

//listen 监听.
func (p *Proxy) listen() {
	listener, err := net.Listen("tcp4", p.Addr)
	if err != nil {
		p.Ch <- false
		log.Printf("err:%+v\n", err)
		return
	}
	p.Ch <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("err:%+v\n", err)
			return
		}

		client := p.NewClient(conn, p.connServer())
		go p.handler(client)
	}
}

//handler 处理请求.
func (p *Proxy) handler(cli *Client) {
	redis := p.NewReais(cli)
	for {
		//解析redis协议.
		proto, err := redis.Parse()
		if err != nil {
			if err == io.EOF {
				log.Println(cli.IP, " 断开连接")
				return
			} else {
				log.Printf("err:%+v\n", err)
				return
			}
		}

		switch proto.Command {
		case "COMMAND":
			redis.connection()
		case "PING":
			redis.ping()
		case "SET":
			redis.set(cli.ServerConn, proto.Args)
		case "GET":
			redis.get(cli.ServerConn, proto.Args)
		case "DEL":
			redis.del(cli.ServerConn, proto.Args)
		default:
			redis.error("暂不支持当前命令")
		}
	}
}
