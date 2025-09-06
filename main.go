package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/EnnioSimoes/Rate-Limiter/limiter"
	"github.com/EnnioSimoes/Rate-Limiter/middleware"
	"github.com/EnnioSimoes/Rate-Limiter/storage"

	"github.com/joho/godotenv"
)

func main() {
	// Carrega as variáveis de ambiente do arquivo .env
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Aviso: Não foi possível encontrar o arquivo .env, usando variáveis de ambiente do sistema.", err)
	}

	// Configuração do Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	// Inicializa a estratégia de armazenamento (Redis)
	redisStorage := storage.NewRedisStorage(redisAddr, redisPassword, redisDB)

	// Carrega as configurações do limiter a partir do ambiente
	limiterConfig, err := limiter.LoadConfigFromEnv(redisStorage)
	if err != nil {
		log.Fatalf("Erro ao carregar configurações do limiter: %v", err)
	}

	// Cria a instância do Rate Limiter
	rateLimiter := limiter.NewRateLimiter(limiterConfig)

	// Cria o middleware
	rateLimiterMiddleware := middleware.RateLimiterMiddleware(rateLimiter)

	// Configuração das rotas e do servidor
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Bem-vindo! Requisição permitida.\n"))
	})

	// Aplica o middleware ao handler
	handler := rateLimiterMiddleware(mux)

	serverPort := os.Getenv("SERVER_PORT")
	fmt.Printf("Servidor escutando na porta: %s\n", serverPort)
	if err := http.ListenAndServe(":"+serverPort, handler); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
