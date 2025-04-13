package dao

import(
	"gorm.io/gorm"
)
func InitDB(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}