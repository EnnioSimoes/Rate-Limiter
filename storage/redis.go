package storage

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStorage implementa a interface limiter.Storage usando Redis.
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage cria e retorna uma nova instância de RedisStorage.
func NewRedisStorage(addr, password string, db int) *RedisStorage {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisStorage{client: rdb}
}

var ctx = context.Background()

// Incrementa o contador para uma chave específica.
// A chave expira em 1 segundo para contar apenas as requisições por segundo.
func (r *RedisStorage) Increment(key string) (int, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Se a chave foi criada agora, define a expiração para 1 segundo.
	if val == 1 {
		r.client.Expire(ctx, key, time.Second)
	}
	return int(val), nil
}

// Bloqueia uma chave por uma determinada duração.
func (r *RedisStorage) Block(key string, duration time.Duration) error {
	blockKey := "block:" + key
	return r.client.Set(ctx, blockKey, "blocked", duration).Err()
}

// Verifica se uma chave está bloqueada.
func (r *RedisStorage) IsBlocked(key string) (bool, time.Duration, error) {
	blockKey := "block:" + key
	ttl, err := r.client.TTL(ctx, blockKey).Result()
	if err == redis.Nil {
		return false, 0, nil // Chave não existe, não está bloqueada.
	}
	if err != nil {
		return false, 0, err
	}
	return ttl > 0, ttl, nil
}

// Implementação alternativa usando timestamp para bloqueio.
// func (r *RedisStorage) Block(key string, duration time.Duration) error {
// 	blockKey := "block:" + key
// 	timeStamp := time.Now().Add(duration).Unix()
// 	return r.client.Set(ctx, blockKey, timeStamp, -1).Err()
// }

// // Verifica se uma chave está bloqueada.
// func (r *RedisStorage) IsBlocked(key string) (bool, time.Duration, error) {
// 	blockKey := "block:" + key
// 	val, err := r.client.Get(ctx, blockKey).Result()
// 	if err == redis.Nil {
// 		return false, 0, nil // Chave não existe, não está bloqueada.
// 	}
// 	if err != nil {
// 		return false, 0, err
// 	}

// 	duration, _ := strconv.Atoi(os.Getenv("IP_BLOCK_DURATION_MINUTES"))

// 	// convert duration minutes to seconds
// 	blockDurationSeconds := duration * 60

// 	intVal, err := strconv.ParseInt(val, 10, 64)
// 	if err != nil {
// 		return false, 0, err
// 	}
// 	if time.Now().Unix() < (intVal + int64(blockDurationSeconds)) {
// 		return true, time.Until(time.Unix(intVal+int64(blockDurationSeconds), 0)), nil
// 	}

// 	// O bloqueio expirou, vamos removê-lo
// 	r.client.Del(ctx, blockKey)
// 	return false, 0, nil
// }
