package conf

import (
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

var config map[string]string

//Load 加载配置文件.
func Load(path string) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	sarr := strings.Split(string(b), "\n")

	cf := make(map[string]string)
	for _, line := range sarr {
		//忽略掉空行.
		if len(line) < 1 {
			continue
		}

		//忽略掉注释.
		if strings.HasPrefix(line, "#") {
			continue
		}

		//按照=来拆分配置.
		arr := strings.Split(line, "=")
		key := strings.Trim(arr[0], " ")
		val := strings.Trim(arr[1], " ")
		cf[key] = val
	}
	config = cf
}

func GetString(key string) string {
	//判断key是否存在.
	if _, ok := config[key]; ok {
		return config[key]
	}
	return ""
}

func GetInt(key string) int {
	if _, ok := config[key]; ok {
		n, err := strconv.Atoi(config[key])
		if err != nil {
			log.Printf("err:%+v\n", err)
			return 0
		}
		return n
	}
	return 0
}

func GetBool(key string) bool {
	if _, ok := config[key]; ok {
		res, err := strconv.ParseBool(config[key])
		if err != nil {
			log.Printf("err:%+v\n", err)
			return false
		}

		return res
	}
	return true
}
