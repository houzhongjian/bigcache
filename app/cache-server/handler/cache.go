package handler

import (
	"encoding/json"
	"io"
	"log"
	"net"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/houzhongjian/bigcache/lib/errcode"
	"github.com/houzhongjian/bigcache/lib/packet"

	"github.com/houzhongjian/bigcache/lib/conf"
)

type Cache struct {
	Addr    string
	Ch      chan bool
	Storage StorageEngine
}

//NewServer.
func NewServer() Cache {
	cache := Cache{
		Addr:    conf.GetString("addr"),
		Ch:      make(chan bool),
		Storage: NewStorage(conf.GetString("storage_dir")),
	}
	return cache
}

//Start.
func (cache *Cache) Start() {
	cache.start()
}

func (cache *Cache) start() {
	go cache.checkServerStart()
	cache.listen()
}

//checkServerStart 检查是否启动成功.
func (cache *Cache) checkServerStart() {
	for {
		select {
		case msg := <-cache.Ch:
			if msg {
				cache.welcome()
			} else {
				log.Println("启动失败")
			}
		}
	}
}

func (cache *Cache) welcome() {
	log.Println("Bigcache Server")
}

func (cache *Cache) listen() {
	listener, err := net.Listen("tcp4", cache.Addr)
	if err != nil {
		cache.Ch <- false
		log.Printf("err:%+v\n", err)
		return
	}
	cache.Ch <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("err:%+v\n", err)
			return
		}

		client := cache.NewClient(conn)
		go cache.handler(client)
	}
}

func (cache *Cache) handler(cli *Client) {
	for {
		pkt, err := packet.ParseRequest(cli.Conn)
		if err != nil {
			log.Printf("err:%+v\n", err)
			if err == io.EOF {
				log.Println("断开连接!")
				return
			}
			cli.Write(err.Error(), errcode.INFO)
			continue
		}

		switch pkt.Protocol {
		case packet.WRITE:
			cache.Write(pkt.Body, cli)
		case packet.READ:
			cache.Read(pkt.Body, cli)
		case packet.DELETE:
			cache.Delete(pkt.Body, cli)
		}
	}
}

//Read 读操作.
func (cache *Cache) Read(body []byte, cli *Client) {
	//读操作.
	content := []string{}
	if err := json.Unmarshal(body, &content); err != nil {
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}

	key := content[0]

	val, err := cache.Storage.Read(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			cli.Write(err.Error(), errcode.NOT_FOUND)
			return
		}
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}

	cli.Write(val, errcode.NO_ERROR)
}

//Write 写操作
func (cache *Cache) Write(body []byte, cli *Client) {
	//写操作.
	content := []string{}
	if err := json.Unmarshal(body, &content); err != nil {
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}

	key := content[0]
	val := content[1]

	if err := cache.Storage.Write(key, val); err != nil {
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}

	cli.Write("OK", errcode.NO_ERROR)
}

//Delete 删除操作.
func (cache *Cache) Delete(body []byte, cli *Client) {
	//删除操作.
	content := []string{}
	if err := json.Unmarshal(body, &content); err != nil {
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}

	key := content[0]

	err := cache.Storage.Delete(key)
	if err != nil {
		log.Printf("err:%+v\n", err)
		cli.Write(err.Error(), errcode.INFO)
		return
	}

	cli.Write("OK", errcode.NO_ERROR)
}
