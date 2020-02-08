package base

import "net"

//插槽类型.
type SlotType uint

const (
	SLOT_TYPE_NORMAL  SlotType = 1 //正常状态.
	SLOT_TYPE_MIGRATE SlotType = 2 //迁移状态.
)

type Slot struct {
	ID       int
	Types    SlotType
	TypeName string   `json:"-"`
	IP       string   //插槽对应的ip地址.
	NewIP    string   //当插槽处于迁移状态的时候，当前属性才会有值.
	Conn     net.Conn `json:"-"`
	NewConn  net.Conn `json:"-"`
}

func SwitchSlotType(types SlotType) string {
	var name string
	switch types {
	case SLOT_TYPE_NORMAL:
		name = "正常"
	case SLOT_TYPE_MIGRATE:
		name = "迁移"
	}

	return name
}
