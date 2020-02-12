package handler

import (
	"log"
	"strings"
	"time"

	"github.com/houzhongjian/bigcache/lib/conf"
	"go.etcd.io/etcd/clientv3"
)

func newEtcd() *clientv3.Client {
	sarr := strings.Split(conf.GetString("etcd_addr"), ",")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   sarr,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Panicf("err:%+v\n", err)
		return nil
	}

	return cli
}
