package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
)

func main() {
	send()
}

func read() {
	opt := &redis.Options{
		Addr:     "127.0.0.1:6378",
		Password: "",
		DB:       0,
	}

	client := redis.NewClient(opt)
	pong, err := client.Ping().Result()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}
	log.Println(pong)

	content := client.Get("article_123").String()
	log.Println(content)
}

func send() {
	opt := &redis.Options{
		Addr:     "127.0.0.1:6378",
		Password: "",
		DB:       0,
	}

	client := redis.NewClient(opt)
	pong, err := client.Ping().Result()
	if err != nil {
		log.Printf("err:%+v\n", err)
		return
	}
	log.Println(pong)

	st := time.Now().Unix()
	log.Println("开始时间:", st)
	for i := 1; i <= 100000; i++ {
		key := fmt.Sprintf("article_%d", i)
		val := key

		err = client.Set(key, val, 0).Err()
		if err != nil {
			log.Printf("err:%+v\n", err)
			return
		}
	}
	et := time.Now().Unix()
	log.Println("结束时间:", et)

	log.Println("总共用时:", et-st)
}
