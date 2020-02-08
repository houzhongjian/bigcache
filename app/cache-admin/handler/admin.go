package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"

	"github.com/gin-gonic/gin"

	"github.com/houzhongjian/bigcache/base"
	"github.com/houzhongjian/bigcache/lib/conf"
	"github.com/houzhongjian/bigcache/lib/utils"
)

type Admin struct {
	Addr string
	Etcd *clientv3.Client
}

type AdminEngine interface {
	Start()
	newWeb()
}

type SlotParams struct {
	StartSlot int
	EndSlot   int
	IP        string
}

//插槽类型.
type SlotType uint

const (
	SLOT_TYPE_NORMAL  SlotType = 1 //正常状态.
	SLOT_TYPE_MIGRATE SlotType = 2 //迁移状态.
)

type Slot struct {
	ID      uint
	Types   SlotType
	IP      string   //插槽对应的ip地址.
	NewIP   string   //当插槽处于迁移状态的时候，当前属性才会有值.
	Conn    net.Conn `json:"-"`
	NewConn net.Conn `json:"-"`
}

//NewAdmin 返回一个Admin接口.
func NewAdmin() AdminEngine {
	var admin AdminEngine
	admin = &Admin{
		Addr: fmt.Sprintf(":%s", conf.GetString("addr")),
		Etcd: newEtcd(),
	}
	return admin
}

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

func (admin *Admin) Start() {
	admin.newWeb()
}

//newWeb .
func (admin *Admin) newWeb() {
	r := gin.New()
	r.Static("/static/", "./static/")
	r.LoadHTMLGlob("./web/*")
	r.GET("/admin/index", admin.IndexHandle)
	r.GET("/admin/node", admin.NodeHandle)
	r.POST("/admin/node", admin.NodeHandle)
	r.GET("/admin/slot", admin.SlotHandle)
	r.POST("/admin/slot", admin.SlotHandle)
	r.GET("/admin/migrate", admin.MigrateHandle)
	r.POST("/admin/migrate", admin.MigrateHandle)
	// r.POST("/admin/slot", admin.SlotHandle)
	r.Run(admin.Addr)
}

func (admin *Admin) ReturnJson(c *gin.Context, msg string, status bool) {
	dist := map[string]interface{}{
		"msg":    msg,
		"status": status,
	}

	c.JSON(http.StatusOK, dist)
}

func (admin *Admin) IndexHandle(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

//NodeHandle 节点.
func (admin *Admin) NodeHandle(c *gin.Context) {
	if c.Request.Method == "POST" {
		serverId := c.PostForm("serverId")
		if len(serverId) < 1 {
			admin.ReturnJson(c, "编号不能为空", false)
			return
		}
		serverIp := c.PostForm("serverIp")
		if len(serverIp) < 1 {
			admin.ReturnJson(c, "ip不能为空", false)
			return
		}

		cacheServer := base.CacheServer{
			ID:    uint(utils.ParseInt(serverId)),
			IP:    serverIp,
			Types: base.CACHESERVER_TYPE_NORMAL,
		}
		b, err := json.Marshal(cacheServer)
		if err != nil {
			log.Printf("err:%+v\n", err)
			admin.ReturnJson(c, "添加失败", false)
			return
		}

		_, err = admin.Etcd.Put(context.Background(), fmt.Sprintf("/cacheserver/%s", serverId), string(b))
		if err != nil {
			log.Printf("err:%+v\n", err)
			admin.ReturnJson(c, "添加失败", false)
			return
		}

		admin.ReturnJson(c, "添加成功", true)
		return
	}

	//获取所有的cache server 节点.
	list, err := admin.getCacheServerList()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	c.HTML(http.StatusOK, "node.html", map[string]interface{}{"CacheServerList": list})
}

//getCacheServerList 获取所有的cache server.
func (admin *Admin) getCacheServerList() (list []base.CacheServer, err error) {
	response, err := admin.Etcd.Get(context.Background(), "/cacheserver/", clientv3.WithPrefix())
	if err != nil {
		log.Printf("err:%+v\n", err)
		return list, err
	}

	for _, v := range response.Kvs {
		value := string(v.Value)
		cacheServer := base.CacheServer{}
		if err := json.Unmarshal([]byte(value), &cacheServer); err != nil {
			log.Printf("err:%+v\n", err)
			return list, err
		}
		cacheServer.TypeName = base.SwitchCacheServerType(cacheServer.Types)

		list = append(list, cacheServer)
	}
	return list, nil
}

//SlotHandle 插槽.
func (admin *Admin) SlotHandle(c *gin.Context) {
	if c.Request.Method == "POST" {
		serverIp := c.PostForm("serverIp")
		if len(serverIp) < 1 {
			admin.ReturnJson(c, "所属ip不能为空", false)
			return
		}
		startSlot := utils.ParseInt(c.PostForm("startSlot"))
		if startSlot < 0 || startSlot > 1023 {
			admin.ReturnJson(c, "插槽信息错误", false)
			return
		}
		endSlot := utils.ParseInt(c.PostForm("endSlot"))
		if startSlot < 0 || endSlot > 1023 {
			admin.ReturnJson(c, "插槽信息错误", false)
			return
		}

		if startSlot > endSlot {
			admin.ReturnJson(c, "插槽信息错误", false)
			return
		}

		for i := startSlot; i <= endSlot; i++ {
			key := fmt.Sprintf("/slot/%d", i)
			// val :=
			slot := base.Slot{
				ID:    i,
				Types: base.SLOT_TYPE_NORMAL,
				IP:    serverIp,
			}

			b, err := json.Marshal(slot)
			if err != nil {
				admin.ReturnJson(c, "设置插槽失败", false)
				return
			}
			_, err = admin.Etcd.Put(context.Background(), key, string(b))
			if err != nil {
				log.Printf("err:%+v\n", err)
				admin.ReturnJson(c, "设置插槽失败", false)
				return
			}
		}

		admin.ReturnJson(c, "设置插槽成功", true)
		return
	}

	cacheServer, err := admin.getCacheServerList()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	//获取所有的插槽信息.
	slotList, err := admin.getSlotList()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	slotData := map[string][]int{}
	for _, v := range slotList {
		//判断当前是否存在.
		if _, ok := slotData[v.IP]; !ok {
			slotData[v.IP] = []int{v.ID}
		} else {
			slotTmpRange := slotData[v.IP]
			slotTmpRange = append(slotTmpRange, v.ID)

			//排序.
			sort.Ints(slotTmpRange)
			slotData[v.IP] = slotTmpRange
		}
	}

	// log.Printf("slotData:%+v\n", slotData)
	data := map[string]interface{}{
		"CacheServerList": cacheServer,
		"SlotList":        slotData,
	}
	c.HTML(http.StatusOK, "slot.html", data)
}

//getSlotList 获取所有的插槽信息.
func (admin *Admin) getSlotList() (list []base.Slot, err error) {
	response, err := admin.Etcd.Get(context.Background(), "/slot/", clientv3.WithPrefix())
	if err != nil {
		log.Printf("err:%+v\n", err)
		return list, err
	}

	for _, v := range response.Kvs {
		value := string(v.Value)
		slot := base.Slot{}
		if err := json.Unmarshal([]byte(value), &slot); err != nil {
			log.Printf("err:%+v\n", err)
			return list, err
		}
		slot.TypeName = base.SwitchSlotType(slot.Types)

		list = append(list, slot)
	}
	return list, nil
}

//MigrateHandle 数据迁移.
func (admin *Admin) MigrateHandle(c *gin.Context) {
	if c.Request.Method == "POST" {
		return
	}

	cacheServer, err := admin.getCacheServerList()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	data := map[string]interface{}{
		"CacheServerList": cacheServer,
	}
	c.HTML(http.StatusOK, "migrate.html", data)
}
