package handler

import (
	"io"
	"log"
	"net"
	"sync"

	"github.com/houzhongjian/bigcache/lib/etcd"

	"go.etcd.io/etcd/clientv3"

	"github.com/houzhongjian/bigcache/lib/conf"
	"github.com/houzhongjian/bigcache/lib/utils"
)

type Proxy struct {
	Addr        string
	Ch          chan bool
	Etcd        *clientv3.Client
	Lock        *sync.RWMutex
	CacheServer map[string]net.Conn
}

//节点类型.
type NodeType uint

const (
	CACHE_PROXY_NODE  NodeType = 1 //代理节点.
	CACHE_SERVER_NODE NodeType = 2 //存储节点.
)

type Node struct {
	Types NodeType
	IP    string
}

//NewProxy.
func NewProxy() Proxy {
	p := Proxy{
		Addr:        conf.GetString("addr"),
		Ch:          make(chan bool),
		Etcd:        etcd.New(conf.GetString("etcd_addr")),
		CacheServer: make(map[string]net.Conn),
		Lock:        &sync.RWMutex{},
	}
	return p
}

func (p *Proxy) Start() {
	p.start()
}

func (p *Proxy) start() {
	//监听tcp端口.
	go p.checkProxyStart()
	// go p.connMasterNode()
	p.listen()
}

// connCacheServer 连接cacheServer 节点.
func (p *Proxy) connCacheServer(ip string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	conn, err := net.Dial("tcp4", ip)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	log.Println("cache server ip:", ip, "连接成功!")
	p.CacheServer[ip] = conn
}

//removeCacheServer 移除cache server.
func (p *Proxy) removeCacheServer(ip string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.CacheServer[ip].Close()
	delete(p.CacheServer, ip)
	log.Println("cache server ip:", ip, "移除成功!")
}

//checkProxyStart 检查是否启动成功.
func (p *Proxy) checkProxyStart() {
	for {
		select {
		case msg := <-p.Ch:
			if msg {
				//打印欢迎界面.
				p.welcome()
				//连接所有的cache server节点.
				p.getCacheServerList()
				//监听是否有新的cache server节点添加.
				p.etcdWatch()
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

		client := p.NewClient(conn)
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

		if proto.Command == "COMMAND" {
			redis.connection()
			continue
		}

		//1、计算当前key的插槽.
		//2、根据插槽去etcd中获取当前插槽对应的ip地址.
		//3、根据ip地址在cacheServer中获取对应的连接地址.

		//1、 计算key对应的插槽
		slot := utils.Slot(string(proto.Args[0]))
		log.Println(slot)

		//2、根据插槽获取对应的ip地址
		ip, err := p.getSlot(slot)
		if err != nil {
			redis.error(err.Error())
			continue
		}

		log.Println("定位cache server ip:", ip)
		cacheServerConn, ok := p.CacheServer[ip]
		if !ok {
			redis.error("服务器错误")
			continue
		}

		switch proto.Command {
		case "PING":
			redis.ping()
		case "SET":
			redis.set(cacheServerConn, proto.Args)
		case "GET":
			redis.get(cacheServerConn, proto.Args)
		case "DEL":
			redis.del(cacheServerConn, proto.Args)
		default:
			redis.error("暂不支持当前命令")
		}
	}
}
