package limiter

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contém as configurações do rate limiter.
type Config struct {
	IPRequestsPerSecond int
	IPBlockDuration     time.Duration
	TokenLimits         map[string]TokenConfig
	Storage             Storage
}

// TokenConfig define o limite e a duração do bloqueio para um token específico.
type TokenConfig struct {
	Limit         int
	BlockDuration time.Duration
}

// RateLimiter é a estrutura principal que gerencia os limites.
type RateLimiter struct {
	config Config
}

// NewRateLimiter cria um novo RateLimiter com as configurações fornecidas.
func NewRateLimiter(config Config) *RateLimiter {
	return &RateLimiter{config: config}
}

// LoadConfigFromEnv carrega as configurações das variáveis de ambiente.
func LoadConfigFromEnv(storage Storage) (Config, error) {
	ipLimit, _ := strconv.Atoi(os.Getenv("IP_REQUESTS_PER_SECOND"))
	ipBlock, _ := strconv.Atoi(os.Getenv("IP_BLOCK_DURATION_MINUTES"))

	config := Config{
		IPRequestsPerSecond: ipLimit,
		IPBlockDuration:     time.Duration(ipBlock) * time.Minute,
		TokenLimits:         make(map[string]TokenConfig),
		Storage:             storage,
	}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key, value := pair[0], pair[1]

		if strings.HasPrefix(key, "TOKEN_LIMIT_") {
			token := strings.TrimPrefix(key, "TOKEN_LIMIT_")

			fmt.Println("Configurando token:", token, "com valor:", value)

			parts := strings.Split(value, ",")
			if len(parts) == 2 {
				limit, _ := strconv.Atoi(parts[0])
				blockDuration, _ := strconv.Atoi(parts[1])
				config.TokenLimits[token] = TokenConfig{
					Limit:         limit,
					BlockDuration: time.Duration(blockDuration) * time.Minute,
				}
			}
		}
	}

	return config, nil
}

// Allow verifica se uma requisição de um determinado identificador (IP ou token) é permitida.
func (rl *RateLimiter) Allow(identifier string) bool {
	// 1. Verifica se o identificador já está bloqueado
	blocked, _, err := rl.config.Storage.IsBlocked(identifier)
	if err != nil {
		fmt.Printf("Erro ao verificar bloqueio para %s: %v\n", identifier, err)
		return false // Em caso de erro, bloqueia por segurança
	}
	if blocked {
		return false
	}

	// 2. Define o limite e a duração do bloqueio com base no tipo de identificador
	limit := rl.config.IPRequestsPerSecond
	blockDuration := rl.config.IPBlockDuration

	if tokenConfig, isToken := rl.config.TokenLimits[identifier]; isToken {
		limit = tokenConfig.Limit
		blockDuration = tokenConfig.BlockDuration
	}

	// 3. Incrementa o contador de requisições
	count, err := rl.config.Storage.Increment(identifier)
	if err != nil {
		fmt.Printf("Erro ao incrementar contador para %s: %v\n", identifier, err)
		return false // Bloqueia em caso de erro
	}

	// 4. Verifica se o limite foi excedido
	if count > limit {
		// 5. Se excedeu, bloqueia o identificador
		fmt.Printf("Limite excedido para %s. Bloqueando por %v\n", identifier, blockDuration)
		rl.config.Storage.Block(identifier, blockDuration)
		return false
	}

	return true
}
