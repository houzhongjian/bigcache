package migrate

import (
	"log"
	"time"

	"github.com/houzhongjian/bigcache/base"
)

type Migrate struct {
}

//List 迁移数据的列表
var List = make(chan base.Slot, 1024)

func New() *Migrate {
	return &Migrate{}
}

//Start 开启迁移.
func (m *Migrate) Start() {
	log.Println("执行迁移任务!")
	m.migrate()
}

func (m *Migrate) migrate() {
	for {
		select {
		case slot := <-List:
			m.handler(slot)
		}
	}
}

func (m *Migrate) handler(slot base.Slot) {
	log.Println("开始迁移:", slot.ID)
	time.Sleep(time.Second * 5)
	log.Println(slot.ID, "迁移完成")
}
