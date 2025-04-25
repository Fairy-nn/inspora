package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

var ErrKeyNotExist = redis.Nil

type UserCacheInterface interface {
	Get(ctx context.Context, id int64) (domain.User, error)
	Set(ctx context.Context, user domain.User) error
	Key(id int64) string
}

// client 可以接收单机redis客户端或集群redis客户端
// expiation 缓存的过期时间
type UserCache struct {
	client     redis.Cmdable
	expiration time.Duration
}

// 接受一个redis服务器地址addr
// expiation 设置为15分钟
func NewUserCacheV1(client redis.Cmdable) UserCacheInterface {
	// client := redis.NewClient(&redis.Options{})
	return &UserCache{
		client:     client,
		expiration: time.Minute * 15,
	}
}

func (cache *UserCache) Key(id int64) string {
	return fmt.Sprintf("user:info:%d", id)
}

// get方法用于从redis缓存中获取用户信息
func (c *UserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	key := c.Key(id) // 生成缓存的key
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == ErrKeyNotExist {
			return domain.User{}, ErrKeyNotExist
		}
		return domain.User{}, err
	}
	var user domain.User
	err = json.Unmarshal(val, &user)
	return user, err
}

// set方法用于将用户信息存入redis缓存
func (c *UserCache) Set(ctx context.Context, user domain.User) error {
	val, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.Key(user.ID), val, c.expiration).Err()
}
