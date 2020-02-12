package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"

	"go.etcd.io/etcd/clientv3"

	"github.com/gin-gonic/gin"

	"github.com/houzhongjian/bigcache/app/cache-admin/model"
	"github.com/houzhongjian/bigcache/base"
	"github.com/houzhongjian/bigcache/lib/conf"
	"github.com/houzhongjian/bigcache/lib/utils"
)

type Admin struct {
	Addr  string
	Etcd  *clientv3.Client
	Model *model.Model
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

//NewAdmin 返回一个Admin接口.
func NewAdmin() AdminEngine {
	var admin AdminEngine
	admin = &Admin{
		Addr:  fmt.Sprintf(":%s", conf.GetString("addr")),
		Etcd:  newEtcd(),
		Model: model.New(),
	}
	return admin
}

func (admin *Admin) Start() {
	admin.newWeb()
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
		slotid := utils.ParseInt(c.PostForm("slotid"))
		if slotid < 0 || slotid > 1023 {
			admin.ReturnJson(c, "插槽id错误", false)
			return
		}

		//更具插槽id获取当前插槽所属服务器.
		slot, err := admin.getIPbySlotid(slotid)
		if err != nil {
			log.Printf("err:%+v\n", err)
			admin.ReturnJson(c, "创建任务失败", false)
			return
		}

		//判断当前插槽的状态.
		if slot.Types == base.SLOT_TYPE_MIGRATE {
			admin.ReturnJson(c, "当前插槽已经处于迁移状态", false)
			return
		}

		targetip := c.PostForm("targetip")
		if len(targetip) < 1 {
			admin.ReturnJson(c, "目标ip不能为空", false)
			return
		}

		//判断目标ip是否与原ip一致.
		if targetip == slot.IP {
			admin.ReturnJson(c, "目标ip不能与原ip一致", false)
			return
		}

		task := model.Task{
			SlotID:    slotid,
			MigrateIP: slot.IP,
			TargetIP:  targetip,
			Status:    0,
		}
		if err := admin.Model.CreateTask(task); err != nil {
			log.Printf("err:%+v\n", err)
			admin.ReturnJson(c, "创建任务失败", false)
			return
		}

		admin.ReturnJson(c, "创建任务成功", true)
		return
	}

	cacheServer, err := admin.getCacheServerList()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	//获取任务.
	task, err := admin.Model.GetTaskList()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}

	data := map[string]interface{}{
		"CacheServerList": cacheServer,
		"TaskList":        task,
	}
	c.HTML(http.StatusOK, "migrate.html", data)
}

func (admin *Admin) getIPbySlotid(slotid int) (slot base.Slot, err error) {
	response, err := admin.Etcd.Get(context.Background(), fmt.Sprintf("/slot/%d", slotid))
	if err != nil {
		log.Printf("err:%+v\n", err)
		return slot, err
	}

	var data []byte
	for _, item := range response.Kvs {
		data = item.Value
	}

	if err := json.Unmarshal(data, &slot); err != nil {
		log.Printf("err:%+v\n", err)
		return slot, err
	}

	return slot, nil
}

//StartMigrateHandle .
func (admin *Admin) StartMigrateHandle(c *gin.Context) {
	taskid := utils.ParseInt(c.PostForm("taskid"))
	if taskid < 1 {
		admin.ReturnJson(c, "请求数据错误", false)
		return
	}

	//获取任务信息.
	task, err := admin.Model.QueryTaskById(taskid)
	if err != nil {
		log.Printf("err:%+v\n", err)
		admin.ReturnJson(c, err.Error(), false)
		return
	}

	//更改插槽的数据.
	slot := base.Slot{
		ID:    task.SlotID,
		Types: base.SLOT_TYPE_NORMAL,
		IP:    task.TargetIP,
	}
	b, err := json.Marshal(slot)
	if err != nil {
		log.Printf("err:%+v\n", err)
		admin.ReturnJson(c, "请求任务信息失败", false)
		return
	}
	_, err = admin.Etcd.Put(context.Background(), fmt.Sprintf("/slot/%d", task.SlotID), string(b))
	if err != nil {
		log.Printf("err:%+v\n", err)
		admin.ReturnJson(c, "请求任务信息失败", false)
		return
	}

	//更改任务状态.
	if err := admin.Model.UpdateTaskStatusById(taskid, 1); err != nil {
		admin.ReturnJson(c, err.Error(), false)
		return
	}

	//TODO 执行迁移任务.

	admin.ReturnJson(c, "开始迁移", true)
}
