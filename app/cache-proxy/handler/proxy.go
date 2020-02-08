package handler

import (
	"io"
	"log"
	"net"
	"sync"

	"github.com/houzhongjian/bigcache/lib/etcd"

	"go.etcd.io/etcd/clientv3"

	"github.com/houzhongjian/bigcache/lib/conf"
)

type Proxy struct {
	Addr        string
	Ch          chan bool
	Etcd        *clientv3.Client
	Lock        *sync.RWMutex
	CacheServer map[string]net.Conn
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

		//根据key获取插槽信息.
		slot, err := p.getSlot(proto)
		if err != nil {
			redis.error(err.Error())
			continue
		}

		redis.service(proto, slot)
	}
}

//getCacheServerConn 根据key 获取cache server 的连接.
//1、计算当前key的插槽.
//2、根据插槽去etcd中获取当前插槽对应的ip地址.
//3、根据ip地址在cacheServer中获取对应的连接地址.
// func (p *Proxy) getCacheServerIP(key string) (conn net.Conn, err error) {
// 	//1、 计算key对应的插槽
// 	slot := utils.Slot(key)
// 	log.Println(slot)

// 	//2、获取插槽信息
// 	data, err := p.getSlot(slot)
// 	if err != nil {
// 		return nil, err
// 	}
// 	slotInfo := &Slot{}
// 	if err := json.Unmarshal([]byte(data), &slotInfo); err != nil {
// 		log.Printf("err:%+v\n", err)
// 		return nil, err
// 	}

// 	//判断插槽的状态.
// 	if slotInfo.Types == SLOT_TYPE_NORMAL {

// 	}

// 	log.Println("定位cache server ip:", ip)
// 	cacheServerConn, ok := p.CacheServer[ip]
// 	if !ok {
// 		return nil, errors.New("服务器错误")
// 	}

// 	return cacheServerConn, nil
// }
