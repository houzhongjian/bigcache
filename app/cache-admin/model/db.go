package model

import (
	"fmt"

	"github.com/houzhongjian/bigcache/lib/conf"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Model struct {
	db *gorm.DB
}

type ModelConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func New() *Model {
	m := &Model{}
	m.connection()
	return m
}

//连接数据库.
func (m *Model) connection() *gorm.Model {
	cf := ModelConfig{
		Host:     conf.GetString("db_host"),
		Port:     conf.GetInt("db_port"),
		User:     conf.GetString("db_user"),
		Password: conf.GetString("db_password"),
		Name:     conf.GetString("db_name"),
	}

	args := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		cf.User,
		cf.Password,
		cf.Host,
		cf.Port,
		cf.Name,
	)
	db, err := gorm.Open("mysql", args)
	if err != nil {
		panic(err)
	}

	// 启用Logger，显示详细日志
	db.LogMode(true)
	m.db = db

	m.AutoMigrate()
	return nil
}

//AutoMigrate 数据库自动迁移.
func (m *Model) AutoMigrate() {
	m.db.AutoMigrate(
		&User{},
		&Task{},
	)
}
