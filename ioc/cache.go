package ioc

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitCache() redis.Cmdable {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var cfg = Config{
		Addr: "redis://localhost:6379/0",
	}
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
	})

	// redisClient := redis.NewClient(&redis.Options{
	// 	Addr: "localhost:6379",
	// })

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}

	return redisClient
}
