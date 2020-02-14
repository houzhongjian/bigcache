package main

import (
	"log"

	"github.com/houzhongjian/bigcache/app/cache-admin/module/handler"
	"github.com/houzhongjian/bigcache/app/cache-admin/module/migrate"
	"github.com/houzhongjian/bigcache/cmd"
	"github.com/houzhongjian/bigcache/lib/conf"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	cmd := cmd.New()
	conf.Load(cmd.Conf)

	//执行迁移.
	mig := migrate.New()
	go mig.Start()

	//开启web控制台.
	srv := handler.NewAdmin()
	srv.Start()
}
