package handler

import (
	"encoding/json"
	"log"
	"net"
	"sync"

	"github.com/houzhongjian/bigcache/lib/errcode"

	"github.com/houzhongjian/bigcache/lib/packet"

	"github.com/houzhongjian/bigcache/lib/conf"
)

type Master struct {
	Addr string
	Ch   chan bool
	Lock *sync.Mutex
	Node map[string]Node
}

type MasterEngine interface {
	Start()
	listen()
	newClient(net.Conn) *Client
	checkMasterStart()
	welcome()
	addNode(body []byte, cli *Client)
	removeNode(body []byte, cli *Client)
	getCacheServerAll(body []byte, cli *Client)
}

//NewMaster 返回一个master接口.
func NewMaster() MasterEngine {
	var master MasterEngine
	master = &Master{
		Addr: conf.GetString("addr"),
		Ch:   make(chan bool),
		Lock: &sync.Mutex{},
		Node: make(map[string]Node),
	}
	return master
}

func (master *Master) Start() {
	master.listen()
}

//checkMasterStart 检查master节点启动是否成功.
func (master *Master) checkMasterStart() {
	for {
		select {
		case msg := <-master.Ch:
			if msg {
				master.welcome()
				return
			}
			log.Println("启动失败")
			close(master.Ch)
		}
	}
}

//welcome 欢迎界面.
func (master *Master) welcome() {
	log.Println("Bigcache Master")
}

func (master *Master) listen() {
	listener, err := net.Listen("tcp4", master.Addr)
	if err != nil {
		master.Ch <- false
		log.Printf("err:%+v\n", err)
		return
	}

	master.Ch <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("err:%+v\n", err)
			return
		}

		client := master.newClient(conn)
		go master.handler(client)
	}
}

func (master *Master) handler(cli *Client) {
	defer cli.Conn.Close()
	for {
		pkt, err := packet.ParseRequest(cli.Conn)
		if err != nil {
			log.Printf("err:%+v\n", err)
			cli.Write(err.Error(), errcode.INFO)
			return
		}

		switch pkt.Protocol {
		case packet.ADD_NODE:
			master.addNode(pkt.Body, cli)
		case packet.GET_CACHE_SERVER_ALL:
			master.getCacheServerAll(pkt.Body, cli)
		}
	}
}

//addNode 上线新节点.
func (master *Master) addNode(body []byte, cli *Client) {
	node := Node{}
	if err := json.Unmarshal(body, &node); err != nil {
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}
	node.Conn = cli.Conn
	log.Printf("node:%+v\n", node)

	master.Node[node.IP] = node
	cli.Write("OK", errcode.NO_ERROR)
}

//removeNode 移除节点.
func (master *Master) removeNode(body []byte, cli *Client) {
	// node := Node{}3 
	// if err := json.Unmarshal(body, &node); err != nil {
	// 	log.Printf("err:%+v\n", err)
	// 	cli.Write(err.Error(), errcode.INFO)
	// 	return
	// }

	// master.Node[node.IP] = node
	// cli.Write("OK", errcode.NO_ERROR)
}

//getCacheServerAll 获取所有的cache server 节点.
func (master *Master) getCacheServerAll(body []byte, cli *Client) {
	nodeList := []Node{}
	for _, v := range master.Node {
		if v.Types == CACHE_SERVER_NODE {
			nodeList = append(nodeList, v)
		}
	}

	b, err := json.Marshal(nodeList)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	cli.Write(string(b), errcode.NO_ERROR)
}
