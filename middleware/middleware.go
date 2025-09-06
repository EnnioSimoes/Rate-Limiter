package middleware

import (
	"net/http"
	"strings"

	"github.com/EnnioSimoes/Rate-Limiter/limiter"
)

// RateLimiterMiddleware é o middleware que aplica o rate limiting.
func RateLimiterMiddleware(limiter *limiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("API_KEY")

			identifier := ""
			// A configuração do token se sobrepõe à do IP
			if token != "" {
				identifier = token
			} else {
				// Pega o IP do cliente. Cuidado com proxies.
				// Para produção, considere "X-Forwarded-For" ou "X-Real-IP".
				ip := strings.Split(r.RemoteAddr, ":")[0]
				identifier = ip
			}

			if !limiter.Allow(identifier) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("you have reached the maximum number of requests or actions allowed within a certain time frame\n"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
