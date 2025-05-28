package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// RedisUplink 实现 uplink.Uplink 接口

type RedisUplink struct {
	Client *redis.Client
	Topic  string
	NameV  string
}

func (r *RedisUplink) Send(data []byte) error {
	return r.Client.Publish(context.Background(), r.Topic, data).Err()
}
func (r *RedisUplink) Name() string { return r.NameV }
func (r *RedisUplink) Type() string { return "redis" }
