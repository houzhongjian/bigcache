package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
		Addr:    fmt.Sprintf(":%d", conf.GetInt("port")),
		Ch:      make(chan bool),
		Storage: NewStorage(conf.GetString("storage_dir")),
	}
	return cache
}

//Start.
func (c *Cache) Start() {
	c.start()
}

func (c *Cache) start() {
	go c.checkServerStart()
	c.listen()
}

//checkServerStart 检查是否启动成功.
func (c *Cache) checkServerStart() {
	for {
		select {
		case msg := <-c.Ch:
			if msg {
				c.welcome()
			} else {
				log.Println("启动失败")
			}
		}
	}
}

func (c *Cache) welcome() {
	b, err := ioutil.ReadFile("./app/cache-server/README")
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}
	log.Println(string(b))
}

func (c *Cache) listen() {
	listener, err := net.Listen("tcp4", c.Addr)
	if err != nil {
		c.Ch <- false
		log.Printf("err:%+v\n", err)
		return
	}
	c.Ch <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("err:%+v\n", err)
			return
		}

		client := c.NewClient(conn)
		go c.handler(client)
	}
}

func (c *Cache) handler(cli *Client) {
	for {
		pkt, err := packet.ParseRequest(cli.Conn)
		if err != nil {
			log.Printf("err:%+v\n", err)
			if err == io.EOF {
				log.Println("断开连接!")
				return
			}

			buf := packet.NewResponse(err.Error(), errcode.INFO)
			cli.Conn.Write(buf)
			continue
		}

		if pkt.Protocol == packet.WRITE {
			//写操作.
			content := []string{}
			if err := json.Unmarshal(pkt.Body, &content); err != nil {
				log.Printf("err:%+v\n", err)
				buf := packet.NewResponse(err.Error(), errcode.INFO)
				cli.Conn.Write(buf)
				return
			}

			key := content[0]
			val := content[1]

			// log.Println(key, val)
			if err := c.Storage.Write(key, val); err != nil {
				log.Printf("err:%+v\n", err)
				buf := packet.NewResponse(err.Error(), errcode.INFO)
				cli.Conn.Write(buf)
				continue
			}

			buf := packet.NewResponse("OK", errcode.NO_ERROR)
			cli.Conn.Write(buf)
		}

		if pkt.Protocol == packet.READ {
			//读操作.
			content := []string{}
			if err := json.Unmarshal(pkt.Body, &content); err != nil {
				log.Printf("err:%+v\n", err)
				buf := packet.NewResponse(err.Error(), errcode.INFO)
				cli.Conn.Write(buf)
				continue
			}

			key := content[0]

			val, err := c.Storage.Read(key)
			if err != nil {
				if err == leveldb.ErrNotFound {
					buf := packet.NewResponse(err.Error(), errcode.NOT_FOUND)
					cli.Conn.Write(buf)
					continue
				}
				log.Printf("err:%+v\n", err)
				buf := packet.NewResponse(err.Error(), errcode.INFO)
				cli.Conn.Write(buf)
				continue
			}

			buf := packet.NewResponse(val, errcode.NO_ERROR)
			cli.Conn.Write(buf)
		}

		if pkt.Protocol == packet.DELETE {
			//读操作.
			content := []string{}
			if err := json.Unmarshal(pkt.Body, &content); err != nil {
				log.Printf("err:%+v\n", err)
				buf := packet.NewResponse(err.Error(), errcode.INFO)
				cli.Conn.Write(buf)
				continue
			}

			key := content[0]

			err := c.Storage.Delete(key)
			if err != nil {
				log.Printf("err:%+v\n", err)
				buf := packet.NewResponse(err.Error(), errcode.INFO)
				cli.Conn.Write(buf)
				continue
			}

			buf := packet.NewResponse("OK", errcode.NO_ERROR)
			cli.Conn.Write(buf)
		}
	}
}
