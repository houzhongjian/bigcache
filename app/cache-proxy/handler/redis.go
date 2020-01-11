package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/houzhongjian/bigcache/lib/errcode"

	"github.com/houzhongjian/bigcache/lib/packet"
)

type Redis struct {
	reader *bufio.Reader
	conn   net.Conn
}

type RedisEngine interface {
}
type RedisProto struct {
	Command string
	Args    [][]byte
}

func (p *Proxy) NewReais(cli *Client) *Redis {
	r := &Redis{
		reader: cli.Reader,
		conn:   cli.Conn,
	}
	return r
}

const (
	//支持单个key存储最大512MB的数据.
	MaxBulkBytesLen = 1024 * 1024 * 512
)

const (
	TypeString    = '+'
	TypeError     = '-'
	TypeInt       = ':'
	TypeBulkBytes = '$'
	TypeArray     = '*'
)

//解析redis协议.
func (r *Redis) Parse() (proto RedisProto, err error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		log.Printf("err:%+v\n", err)
		return proto, err
	}

	if line[0] == '*' {
		var argLength int
		if _, err := fmt.Sscanf(line, "*%d\r\n", &argLength); err != nil {
			log.Printf("err:%+v\n", err)
			return proto, err
		}

		//获取command.
		b, err := r.read()
		if err != nil {
			log.Printf("err:%+v\n", err)
			return proto, err
		}
		proto.Command = strings.ToUpper(string(b))

		//获取具体参数.
		arguments := make([][]byte, argLength-1)
		for i := 0; i < argLength-1; i++ {
			if arguments[i], err = r.read(); err != nil {
				log.Printf("err:%+v\n", err)
				return proto, err
			}
		}

		proto.Args = arguments

		return proto, nil
	}
	return proto, errors.New("参数获取异常")
}

func (r *Redis) read() (b []byte, err error) {
	line, err := r.reader.ReadString('\n')
	var argLength int
	_, err = fmt.Sscanf(line, "$%d\r\n", &argLength)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return b, err
	}

	var buf = make([]byte, argLength)
	_, err = io.ReadFull(r.reader, buf)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return b, err
	}

	r.reader.ReadByte()
	r.reader.ReadByte()
	return buf, nil
}

//connection 连接成功.
func (r *Redis) connection() {
	r.conn.Write([]byte("+OK\r\n"))
}

//.ping
func (r *Redis) ping() {
	r.conn.Write([]byte("+PONG\r\n"))
}

func (r *Redis) error(msg string) {
	r.conn.Write([]byte(fmt.Sprintf("-%s\r\n", msg)))
}

func (r *Redis) write(msg string, l int) {
	msg = fmt.Sprintf("$%d\r\n%s\r\n", l, msg)
	if l < 0 {
		//兼容不存在的key.
		//不存在的key 返回nil.
		msg = fmt.Sprintf("$%d\r\n", l)
	}
	r.conn.Write([]byte(msg))
}

func (r *Redis) int(n int) {
	msg := fmt.Sprintf(":%d\r\n", n)
	r.conn.Write([]byte(msg))
}
func (r *Redis) set(srv net.Conn, args [][]byte) {
	key := string(args[0])
	val := string(args[1])

	content := []string{
		key,
		val,
	}
	b, err := json.Marshal(content)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	buf := packet.NewRequest(b, packet.WRITE)
	_, err = srv.Write([]byte(buf))
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	pkt, err := packet.ParseResponse(srv)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	if pkt.Err != errcode.NO_ERROR {
		r.error(pkt.Msg)
		return
	}
	r.connection()
}

func (r *Redis) get(srv net.Conn, args [][]byte) {
	key := string(args[0])
	content := []string{
		key,
	}
	b, err := json.Marshal(content)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	buf := packet.NewRequest(b, packet.READ)
	_, err = srv.Write([]byte(buf))
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	pkt, err := packet.ParseResponse(srv)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	if pkt.Err != errcode.NO_ERROR {
		if pkt.Err == errcode.NOT_FOUND {
			r.write(pkt.Msg, -1)
			return
		}
		r.error(pkt.Msg)
		return
	}
	r.write(pkt.Msg, len(pkt.Msg))
}

func (r *Redis) del(srv net.Conn, args [][]byte) {
	key := string(args[0])
	content := []string{
		key,
	}
	b, err := json.Marshal(content)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	buf := packet.NewRequest(b, packet.DELETE)
	_, err = srv.Write([]byte(buf))
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	pkt, err := packet.ParseResponse(srv)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	if pkt.Err != errcode.NO_ERROR {
		r.error(pkt.Msg)
		return
	}
	r.int(1)
}
