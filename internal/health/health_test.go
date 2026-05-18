package health

import (
	"testing"
	"time"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func TestScoreHealthyExchange(t *testing.T) {
	got := Score(testHealthConfig(), exchange.ExchangeHealth{
		Exchange:           "Binance",
		WebSocketEnabled:   true,
		WebSocketConnected: true,
		DataFresh:          true,
		LastMessageTime:    time.Now(),
	})
	if got.Score != 100 || got.Status != "ok" {
		t.Fatalf("expected healthy score 100/ok, got %d/%s", got.Score, got.Status)
	}
}

func TestScorePenalties(t *testing.T) {
	got := Score(testHealthConfig(), exchange.ExchangeHealth{
		Exchange:            "Kraken",
		WebSocketEnabled:    true,
		WebSocketConnected:  false,
		RestFallbackActive:  true,
		DataFresh:           false,
		ReconnectCount:      3,
		StaleOrderBookCount: 1,
	})
	if got.Score != 64 {
		t.Fatalf("expected score 64, got %d", got.Score)
	}
	if got.Status != "stale" {
		t.Fatalf("expected stale status, got %s", got.Status)
	}
}

func TestClampScore(t *testing.T) {
	if ClampScore(-10) != 0 {
		t.Fatal("expected lower clamp")
	}
	if ClampScore(120) != 100 {
		t.Fatal("expected upper clamp")
	}
}

func testHealthConfig() config.HealthConfig {
	return config.HealthConfig{
		ScoringEnabled:      true,
		StalePenalty:        20,
		DisconnectedPenalty: 40,
		RestFallbackPenalty: 10,
		ReconnectPenalty:    2,
	}
}
