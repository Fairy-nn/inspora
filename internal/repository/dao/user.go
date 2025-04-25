package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// User 用户结构体,直接对应于数据库表
type User struct {
	ID       int64          `gorm:"primaryKey,autoIncrement"` // 主键
	Email    sql.NullString `gorm:"type:varchar(100);unique"` // 邮箱，唯一索引
	Username string         `gorm:"type:varchar(100)"`        // 用户名
	Password string         `gorm:"type:varchar(100)"`
	Ctime    int64          `gorm:"autoCreateTime"`
	Utime    int64          `gorm:"autoUpdateTime"`
	Phone    sql.NullString `gorm:"type:varchar(20);unique"` // 手机号，唯一索引会冲突,所以允许可以为空
}

// 在这里添加其他字段，例如用户名、头像等
type UserInfo struct {
}

type UserDaoInterface interface {
	GetByPhone(ctx context.Context, phone string) (User, error)
	GetByID(ctx context.Context, id int64) (User, error)
	Insert(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type UserDAO struct {
	db *gorm.DB // 数据库连接对象
}

// NewUserDAO 创建用户数据访问对象
func NewUserDAO(db *gorm.DB) UserDaoInterface {
	return &UserDAO{
		db: db,
	}
}

// 自定义错误
var (
	ErrUserDuplicateEmail = errors.New("用户邮箱已存在")
	ErrUserNotFound       = errors.New("用户不存在")
)

// Insert 创建用户
func (u *UserDAO) Insert(ctx context.Context, user *User) error {
	now := time.Now().UnixMilli()
	user.Ctime = now
	user.Utime = now
	if user.Username == "" {
		user.Username = "未设置昵称" // 如果没有提供用户名，则默认使用“未设置昵称”为用户名
	}
	err := u.db.WithContext(ctx).Create(user).Error // 插入用户数据
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const duplicateEntryCode = 1062 // MySQL 错误代码 1062 表示重复条目
		if mysqlErr.Number == duplicateEntryCode {
			// 邮箱冲突
			return ErrUserDuplicateEmail //自定义错误
		}
	}
	return err
}

// GetByEmail 根据邮箱查找用户
func (u *UserDAO) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := u.db.WithContext(ctx).Where("email = ?", email).First(user).Error // 查找用户
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound // 用户不存在
		}
		return nil, err // 其他错误
	}
	return user, nil // 返回找到的用户
}

// GetByID 根据用户ID获取用户信息
func (ud *UserDAO) GetByID(ctx context.Context, id int64) (User, error) {
	var user User
	err := ud.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return User{}, err
	}
	return user, nil
}

// GetByPhone 根据手机号获取用户信息
func (ud *UserDAO) GetByPhone(ctx context.Context, phone string) (User, error) {
	var user User
	err := ud.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return user, nil
}
