FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rate-limiter-server .

# ------------------------------------
# Estágio 2: Final (Produção)
# ------------------------------------
FROM alpine:latest

WORKDIR /app

# Copia APENAS o executável compilado do estágio de build
COPY --from=builder /app/rate-limiter-server .

# Copia o arquivo .env se ele for necessário para a execução
COPY .env .

# Expõe a porta que a aplicação usa
EXPOSE 8080

# Define o comando para EXECUTAR O ARQUIVO binário
CMD ["./rate-limiter-server"]