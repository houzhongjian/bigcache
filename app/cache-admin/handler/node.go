package handler

import "net"

//节点类型.
type NodeType uint

const (
	CACHE_PROXY_NODE  NodeType = 1 //代理节点.
	CACHE_SERVER_NODE NodeType = 2 //存储节点.
)

type Node struct {
	Types NodeType
	Conn  net.Conn
	IP    string
}
