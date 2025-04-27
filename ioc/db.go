package ioc

import (
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	// db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/inspora"))
	// if err != nil {
	// 	panic("连接数据库失败")
	// }
	type Config struct {
		dsn string `yaml:"dsn"`
	}
	var cfg = Config{
		dsn: "root:root@tcp(localhost:3306)/inspora",
	}
	err := viper.UnmarshalKey("mysql", &cfg)
	if err != nil {
		panic(err)
	}
	db, err := gorm.Open(mysql.Open(cfg.dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	err = dao.InitDB(db) // 初始化数据库
	if err != nil {
		panic("数据库初始化失败")
	}
	return db
}
