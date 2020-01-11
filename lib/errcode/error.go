package errcode

type BigcacheError int

const (
	NO_ERROR  BigcacheError = 1000 //没有错误.
	INFO      BigcacheError = 1001 //普通错误.
	NOT_FOUND BigcacheError = 1002 //数据不存在.
)
