package etcd

import (
	"log"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
)

func New(addr string) *clientv3.Client {
	sarr := strings.Split(addr, ",")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   sarr,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Printf("err:%+v\n", err)
		return nil
	}

	return cli
}
