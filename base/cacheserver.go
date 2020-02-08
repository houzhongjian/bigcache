package base

//CacheServer类型.
type CacheServerType uint

const (
	CACHESERVER_TYPE_NORMAL  CacheServerType = 1 //正常状态.
	CACHESERVER_TYPE_OFFLINE CacheServerType = 2 //离线.
	CACHESERVER_TYPE_MIGRATE CacheServerType = 3 //迁移.
)

type CacheServer struct {
	ID       uint
	Types    CacheServerType
	TypeName string `json:"-"`
	IP       string //对应的ip地址.
	// Slot     [2]int //插槽范围.
}

func SwitchCacheServerType(types CacheServerType) string {
	var name string
	switch types {
	case CACHESERVER_TYPE_NORMAL:
		name = "正常"
	case CACHESERVER_TYPE_OFFLINE:
		name = "离线"
	case CACHESERVER_TYPE_MIGRATE:
		name = "迁移"
	}

	return name
}
