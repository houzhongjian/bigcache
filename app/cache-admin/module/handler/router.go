package handler

import "github.com/gin-gonic/gin"

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
	r.POST("/admin/startmig", admin.StartMigrateHandle)
	r.Run(admin.Addr)
}
