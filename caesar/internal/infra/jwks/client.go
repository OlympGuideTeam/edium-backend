package jwks

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

// Client загружает публичные RSA-ключи с JWKS-эндпоинта и обновляет их раз в час.
type Client struct {
	endpoint string
	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		keys:     make(map[string]*rsa.PublicKey),
	}
}

// Load выполняет начальную загрузку ключей. Должен быть вызван до старта сервера.
func (c *Client) Load(ctx context.Context) error {
	keys, err := c.fetchKeys(ctx)
	if err != nil {
		return fmt.Errorf("jwks load: %w", err)
	}
	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()
	return nil
}

// StartRefresh запускает фоновое обновление ключей каждый час.
func (c *Client) StartRefresh(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.Load(ctx); err != nil {
					slog.ErrorContext(ctx, "jwks: не удалось обновить ключи", "err", err)
				} else {
					slog.InfoContext(ctx, "jwks: ключи обновлены")
				}
			}
		}
	}()
}

// GetKey возвращает RSA-ключ по kid.
func (c *Client) GetKey(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	return k, ok
}

func (c *Client) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("создание запроса: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS вернул статус %d", resp.StatusCode)
	}

	var body jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("декодирование JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(body.Keys))
	for _, k := range body.Keys {
		key, err := parseRSAKey(k)
		if err != nil {
			return nil, fmt.Errorf("парсинг ключа %s: %w", k.Kid, err)
		}
		keys[k.Kid] = key
	}
	return keys, nil
}

func parseRSAKey(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}
