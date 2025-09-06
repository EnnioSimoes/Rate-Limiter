package limiter

import "time"

// Storage é a interface que define os métodos para armazenamento e consulta
// das informações do rate limiter.
type Storage interface {
	// Incrementa o contador para uma chave e retorna o valor atual.
	Increment(key string) (int, error)
	// Bloqueia uma chave por uma determinada duração.
	Block(key string, duration time.Duration) error
	// Verifica se uma chave está bloqueada e retorna o tempo restante de bloqueio.
	IsBlocked(key string) (bool, time.Duration, error)
}
