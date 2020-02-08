package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/houzhongjian/bigcache/base"
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

//先读取新节点.
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

//getMigrate 处理迁移状态的get命令.
//如果插槽处于迁移状态下，先读取新迁移的节点，如果没读取到，在读取旧的节点.
func (r *Redis) getMigrate(srv, newSrv net.Conn, args [][]byte) {
	//读取新节点.
	// newSrv.
	//新节点如果没有读取到，在读取旧节点.
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
	_, err = newSrv.Write([]byte(buf))
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
			//新节点没有读取到数据，现在读取旧节点数据.
			r.get(srv, args)
			return
		}
		r.error(pkt.Msg)
		return
	}
	r.write(pkt.Msg, len(pkt.Msg))
}

//delMigrate 插槽处于迁移状态下的删除操作.
func (r *Redis) delMigrate(srv, newSrv net.Conn, args [][]byte) {
	//先确定数据存在与那个节点.
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
	_, err = newSrv.Write([]byte(buf))
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
			//新节点没有读取到数据，现在读取旧节点数据.
			r.del(srv, args)
			return
		}
		r.error(pkt.Msg)
		return
	}

	r.del(newSrv, args)
}

//selectdb .
func (r *Redis) selectdb(args [][]byte) {
	db := string(args[0])
	_, err := strconv.Atoi(db)
	if err != nil {
		log.Printf("err:%+v\n", err)
		r.error(err.Error())
		return
	}

	r.connection()
}

func (r *Redis) service(proto RedisProto, slot base.Slot) {
	//允许连接.
	if proto.Command == "COMMAND" {
		r.connection()
		return
	}

	//ping.
	if proto.Command == "PING" {
		r.ping()
		return
	}

	if proto.Command == "SELECT" {
		r.selectdb(proto.Args)
		return
	}

	//set 并且当前插槽不处于迁移状态.
	if proto.Command == "SET" && slot.Types == base.SLOT_TYPE_NORMAL {
		r.set(slot.Conn, proto.Args)
		return
	}

	if proto.Command == "SET" && slot.Types == base.SLOT_TYPE_MIGRATE {
		r.set(slot.NewConn, proto.Args)
		return
	}

	if proto.Command == "GET" && slot.Types == base.SLOT_TYPE_NORMAL {
		r.get(slot.Conn, proto.Args)
		return
	}

	if proto.Command == "GET" && slot.Types == base.SLOT_TYPE_MIGRATE {
		r.getMigrate(slot.Conn, slot.NewConn, proto.Args)
		return
	}

	if proto.Command == "DEL" && slot.Types == base.SLOT_TYPE_NORMAL {
		r.del(slot.Conn, proto.Args)
		return
	}

	if proto.Command == "DEL" && slot.Types == base.SLOT_TYPE_MIGRATE {
		r.delMigrate(slot.Conn, slot.NewConn, proto.Args)
		return
	}

	r.error("暂不支持当前命令")
}
