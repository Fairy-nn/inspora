package ioc

import (
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/inspora"))
	if err != nil {
		panic("连接数据库失败")
	}
	err = dao.InitDB(db) // 初始化数据库
	if err != nil {
		panic("数据库初始化失败")
	}
	return db
}
