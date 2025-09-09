package limiter

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockStorage é uma implementação em memória da interface Storage para ser usada em testes.
type mockStorage struct {
	mu     sync.Mutex
	counts map[string]int
	blocks map[string]time.Time
}

// newMockStorage cria uma nova instância do nosso storage em memória.
func newMockStorage() *mockStorage {
	return &mockStorage{
		counts: make(map[string]int),
		blocks: make(map[string]time.Time),
	}
}

// Increment implementa a interface Storage.
func (s *mockStorage) Increment(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[key]++
	return s.counts[key], nil
}

// Block implementa a interface Storage.
func (s *mockStorage) Block(key string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// s.blocks["block:"+key] = time.Now().Add(duration)
	s.blocks["block:"+key] = time.Now().Add(duration)
	return nil
}

// IsBlocked implementa a interface Storage.
func (s *mockStorage) IsBlocked(key string) (bool, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	blockTime, exists := s.blocks["block:"+key]
	if !exists {
		return false, 0, nil
	}

	fmt.Println("TEST - blockTime: ", blockTime)

	if time.Now().Before(blockTime) {
		return true, time.Until(blockTime), nil
	}

	// O bloqueio expirou, vamos removê-lo
	delete(s.blocks, "block:"+key)
	return false, 0, nil
}

// TestRateLimiter_Allow testa os cenários principais de permissão e negação.
func TestRateLimiter_Allow(t *testing.T) {
	// Configuração base para os testes
	baseConfig := Config{
		IPRequestsPerSecond: 5,
		IPBlockDuration:     time.Minute,
		TokenLimits: map[string]TokenConfig{
			"good-token": {Limit: 10, BlockDuration: time.Minute},
		},
	}

	// Tabela de casos de teste
	testCases := []struct {
		name              string
		identifier        string
		requestsToMake    int
		expectedAllowed   int
		expectedFinalDeny bool
	}{
		{
			name:              "IP within limit",
			identifier:        "192.168.1.1",
			requestsToMake:    5,
			expectedAllowed:   5,
			expectedFinalDeny: false,
		},
		{
			name:              "IP exceeding limit",
			identifier:        "192.168.1.2",
			requestsToMake:    6,
			expectedAllowed:   5,
			expectedFinalDeny: true,
		},
		{
			name:              "Token within limit (overrides IP)",
			identifier:        "good-token",
			requestsToMake:    10,
			expectedAllowed:   10,
			expectedFinalDeny: false,
		},
		{
			name:              "Token exceeding limit",
			identifier:        "good-token",
			requestsToMake:    11,
			expectedAllowed:   10,
			expectedFinalDeny: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Cada teste recebe um storage e um limiter novos para garantir o isolamento
			mockStore := newMockStorage()
			config := baseConfig
			config.Storage = mockStore

			limiter := NewRateLimiter(config)

			var allowedCount int
			var lastResult bool
			for i := 0; i < tc.requestsToMake; i++ {
				if limiter.Allow(tc.identifier) {
					allowedCount++
					lastResult = true
				} else {
					lastResult = false
				}
			}

			if allowedCount != tc.expectedAllowed {
				t.Errorf("Expected %d allowed requests, but got %d", tc.expectedAllowed, allowedCount)
			}

			if tc.expectedFinalDeny && lastResult {
				t.Errorf("Expected the last request to be denied, but it was allowed")
			}

			if !tc.expectedFinalDeny && !lastResult {
				t.Errorf("Expected the last request to be allowed, but it was denied")
			}
		})
	}
}

// TestRateLimiter_Blocking testa se o bloqueio é respeitado e se expira corretamente.
func TestRateLimiter_Blocking(t *testing.T) {
	mockStore := newMockStorage()
	config := Config{
		IPRequestsPerSecond: 2,
		IPBlockDuration:     500 * time.Millisecond, // Duração curta para o teste
		Storage:             mockStore,
	}
	limiter := NewRateLimiter(config)

	identifier := "10.0.0.1"

	// 1. Primeira chamada
	if !limiter.Allow(identifier) {
		t.Fatal("Request 1 should be allowed")
	}

	// 2. Segunda chamada
	if !limiter.Allow(identifier) {
		t.Fatal("Request 2 should be allowed")
	}

	// 3. A terceira deve ser bloqueada
	if limiter.Allow(identifier) {
		t.Fatal("Request 3 should be denied as it exceeds the limit")
	}

	// // 4. Espera o tempo de bloqueio passar
	time.Sleep(700 * time.Millisecond)

	// 5. Verifica se o tempo de desbloqueio ja passou
	fmt.Println("identifier: ", identifier)
	// fmt.Println("limiter.Allow(identifier): ", limiter.Allow(identifier))

	// Simula a expiração do contador de requisições, que teria um TTL de 1s no Redis.
	// Como o sleep foi de 700ms, a chave do segundo anterior expirou.
	delete(mockStore.counts, identifier)

	if !limiter.Allow(identifier) {
		t.Fatal("Request should be allowed after the block duration has passed")
	}
}

// TestLoadConfigFromEnv testa o carregamento de configurações das variáveis de ambiente.
func TestLoadConfigFromEnv(t *testing.T) {
	// Define variáveis de ambiente para a duração deste teste
	t.Setenv("IP_REQUESTS_PER_SECOND", "10")
	t.Setenv("IP_BLOCK_DURATION_MINUTES", "2")
	t.Setenv("TOKEN_LIMIT_token1", "100,5")
	t.Setenv("TOKEN_LIMIT_token2", "200,10")

	mockStore := newMockStorage()
	config, err := LoadConfigFromEnv(mockStore)

	if err != nil {
		t.Fatalf("LoadConfigFromEnv failed with error: %v", err)
	}

	if config.IPRequestsPerSecond != 10 {
		t.Errorf("Expected IPRequestsPerSecond to be 10, got %d", config.IPRequestsPerSecond)
	}

	expectedIPBlockDuration := 2 * time.Minute
	if config.IPBlockDuration != expectedIPBlockDuration {
		t.Errorf("Expected IPBlockDuration to be %v, got %v", expectedIPBlockDuration, config.IPBlockDuration)
	}

	if len(config.TokenLimits) != 2 {
		t.Fatalf("Expected to load 2 token limits, but got %d", len(config.TokenLimits))
	}

	// Verifica token1
	token1Config, ok := config.TokenLimits["token1"]
	if !ok {
		t.Fatal("Config for token1 was not loaded")
	}
	if token1Config.Limit != 100 {
		t.Errorf("Expected limit for token1 to be 100, got %d", token1Config.Limit)
	}
	if token1Config.BlockDuration != 5*time.Minute {
		t.Errorf("Expected block duration for token1 to be 5 minutes, got %v", token1Config.BlockDuration)
	}

	// Verifica token2
	token2Config, ok := config.TokenLimits["token2"]
	if !ok {
		t.Fatal("Config for token2 was not loaded")
	}
	if token2Config.Limit != 200 {
		t.Errorf("Expected limit for token2 to be 200, got %d", token2Config.Limit)
	}
	if token2Config.BlockDuration != 10*time.Minute {
		t.Errorf("Expected block duration for token2 to be 10 minutes, got %v", token2Config.BlockDuration)
	}
}
