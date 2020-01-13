package handler

import (
	"context"
	"fmt"
	"log"

	"go.etcd.io/etcd/clientv3"
)

//getCacheServerList 获取所有的cache server.
func (p *Proxy) getCacheServerList() {
	response, err := p.Etcd.Get(context.Background(), "/cacheserver/", clientv3.WithPrefix())
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	for _, v := range response.Kvs {
		cacheServer := string(v.Value)
		p.connCacheServer(cacheServer)
	}
}

//etcdWatch  监听etcd是否有改变.
func (p *Proxy) etcdWatch() {
	for {
		rch := p.Etcd.Watch(context.Background(), "/cacheserver/", clientv3.WithPrefix()) //阻塞在这里，如果没有key里没有变化，就一直停留在这里
		for wresp := range rch {
			for _, ev := range wresp.Events {
				log.Printf("%s %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				cacheServer := string(ev.Kv.Value)

				log.Println("ev.Type:", fmt.Sprintf("%s", ev.Type))
				if fmt.Sprintf("%s", ev.Type) == "PUT" {
					log.Println("新增加cache server 节点")
					p.connCacheServer(cacheServer)
				}

				if fmt.Sprintf("%s", ev.Type) == "DELETE" {
					log.Println("移除cache server 节点")
					p.removeCacheServer(cacheServer)
				}

			}
		}
	}
}

//getSlot 从etcd中根据插槽获取对应的IP地址.
func (p *Proxy) getSlot(slot uint32) (ip string, err error) {
	resp, err := p.Etcd.Get(context.Background(), fmt.Sprintf("/cacheserver/%d", slot))
	if err != nil {
		log.Printf("err:%+v\n", err)
		return ip, err
	}

	for _, item := range resp.Kvs {
		ip = string(item.Value)
		break
	}
	return ip, nil
}
