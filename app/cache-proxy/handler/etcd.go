package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/houzhongjian/bigcache/lib/utils"
	"github.com/houzhongjian/bigcache/base"
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
		value := string(v.Value)
		cacheServer := base.CacheServer{}
		if err := json.Unmarshal([]byte(value), &cacheServer); err != nil {
			log.Printf("err:%+v\n", err)
			return
		}
		log.Printf("cacheServer:%+v\n", cacheServer)
		p.connCacheServer(cacheServer.IP)
	}
}

//etcdWatch  监听cache server 是否有改变.
func (p *Proxy) etcdWatch() {
	for {
		rch := p.Etcd.Watch(context.Background(), "/cacheserver/", clientv3.WithPrefix()) //阻塞在这里，如果没有key里没有变化，就一直停留在这里
		for wresp := range rch {
			for _, ev := range wresp.Events {
				log.Printf("%s %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				value := string(ev.Kv.Value)
				cacheServer := base.CacheServer{}
				if err := json.Unmarshal([]byte(value), &cacheServer); err != nil {
					log.Printf("err:%+v\n", err)
					return
				}

				log.Println("ev.Type:", fmt.Sprintf("%s", ev.Type))
				if fmt.Sprintf("%s", ev.Type) == "PUT" {
					log.Println("新增加cache server 节点")
					p.connCacheServer(cacheServer.IP)
				}

				if fmt.Sprintf("%s", ev.Type) == "DELETE" {
					log.Println("移除cache server 节点")
					p.removeCacheServer(cacheServer.IP)
				}

				if fmt.Sprintf("%s", ev.Type) == "UPDATE" {
					log.Println("更新cache server 节点")
					// p.removeCacheServer(cacheServer)
				}
			}
		}
	}
}

//getSlot 从etcd中根据插槽获取插槽信息.
func (p *Proxy) getSlot(proto RedisProto) (slot base.Slot, err error) {
	if proto.Command == "GET" || proto.Command == "SET" || proto.Command == "DEL" {
		key := string(proto.Args[0])
		slotid := utils.Slot(key)
		log.Println(slotid)

		//从etcd中获取插槽信息.
		resp, err := p.Etcd.Get(context.Background(), fmt.Sprintf("/slot/%d", slotid))
		if err != nil {
			log.Printf("err:%+v\n", err)
			return slot, err
		}

		var data string
		for _, item := range resp.Kvs {
			data = string(item.Value)
		}

		slot = base.Slot{}
		if err := json.Unmarshal([]byte(data), &slot); err != nil {
			log.Printf("err:%+v\n", err)
			log.Printf("data:%+v\n", data)
			return slot, err
		}

		slot.Conn = p.CacheServer[slot.IP]

		if slot.Types == base.SLOT_TYPE_MIGRATE {
			slot.NewConn = p.CacheServer[slot.NewIP]
		}
	}
	return slot, nil

}
