package packet

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"

	"github.com/houzhongjian/bigcache/lib/errcode"
)

const PROROCOL_LEN = 4
const HEADER_LEN = 4

//BigcacheProtocol 传输协议号.
type BigcacheProtocol int

const (
	CONN_AUTH BigcacheProtocol = 1000 //连接授权.
	READ      BigcacheProtocol = 1001 //获取一条记录.
	WRITE     BigcacheProtocol = 1002 //写入一条记录.
	DELETE    BigcacheProtocol = 1003 //删除一条记录.
	MSG       BigcacheProtocol = 1004 //发生一条状态消息.
)

type Request struct {
	Protocol BigcacheProtocol
	Size     int64
	Body     []byte
}

type Response struct {
	Protocol BigcacheProtocol
	Size     int64
	Body     []byte
	Msg      string
	Err      errcode.BigcacheError
}

func NewRequest(content []byte, num BigcacheProtocol) []byte {
	buffer := make([]byte, HEADER_LEN+len(content)+PROROCOL_LEN)
	//0-4 为协议号.
	//4-8 为内容大小.
	//>8 为内容.
	binary.BigEndian.PutUint32(buffer[0:4], uint32(num))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(len(content)))
	copy(buffer[8:], content)
	return buffer
}
func NewResponse(msg string, errcode errcode.BigcacheError) []byte {
	resp := Response{
		Err: errcode,
		Msg: msg,
	}
	buf, err := json.Marshal(resp)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return nil
	}
	// log.Println(string(buf))
	buffer := make([]byte, HEADER_LEN+len(buf)+PROROCOL_LEN)
	//0-4 为协议号.
	//4-8 为内容大小.
	//>8 为内容.
	binary.BigEndian.PutUint32(buffer[0:4], uint32(MSG))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(len(buf)))
	copy(buffer[8:], buf)
	return buffer
}

//ParseRequest 解析网络请求数据包.
func ParseRequest(conn net.Conn) (pkt Request, err error) {
	//获取协议号.
	var num = make([]byte, PROROCOL_LEN)
	_, err = io.ReadFull(conn, num)
	if err != nil {
		return pkt, err
	}
	pkt.Protocol = BigcacheProtocol(int(binary.BigEndian.Uint32(num)))

	//获取内容长度.
	var header = make([]byte, HEADER_LEN)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		return pkt, err
	}
	pkt.Size = int64(binary.BigEndian.Uint32(header))

	//获取内容
	var buf = make([]byte, pkt.Size)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return pkt, err
	}
	pkt.Body = buf

	return pkt, nil
}

//ParseRequest 解析网络返回数据包.
func ParseResponse(conn net.Conn) (pkt Response, err error) {
	//获取协议号.
	var num = make([]byte, PROROCOL_LEN)
	_, err = io.ReadFull(conn, num)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return pkt, err
	}
	pkt.Protocol = BigcacheProtocol(int(binary.BigEndian.Uint32(num)))

	//获取内容长度.
	var header = make([]byte, HEADER_LEN)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return pkt, err
	}
	pkt.Size = int64(binary.BigEndian.Uint32(header))

	//获取内容
	var buf = make([]byte, pkt.Size)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return pkt, err
	}
	pkt.Body = buf
	// log.Println(string(pkt.Body))

	content := Response{}
	if err := json.Unmarshal(pkt.Body, &content); err != nil {
		log.Printf("err:%+v\n", err)
		return pkt, err
	}
	pkt.Err = content.Err
	pkt.Msg = content.Msg

	return pkt, nil
}
