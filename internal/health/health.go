package health

import (
	"time"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func Fresh(lastUpdate time.Time, staleAfter time.Duration) bool {
	if lastUpdate.IsZero() {
		return false
	}
	return time.Since(lastUpdate) <= staleAfter
}

func Score(cfg config.HealthConfig, h exchange.ExchangeHealth) exchange.ExchangeHealth {
	score := 100
	if h.ProviderType == "broker" && h.Enabled && !h.GatewayConnected {
		score -= cfg.DisconnectedPenalty
	}
	if h.WebSocketEnabled && !h.WebSocketConnected && !h.RestFallbackActive {
		score -= cfg.DisconnectedPenalty
	}
	if h.RestFallbackActive {
		score -= cfg.RestFallbackPenalty
	}
	if !h.DataFresh || h.StaleTickerCount > 0 || h.StaleOrderBookCount > 0 {
		score -= cfg.StalePenalty
	}
	if h.LastError != "" {
		score -= cfg.StalePenalty / 2
	}
	score -= h.ReconnectCount * cfg.ReconnectPenalty
	if h.PartialSupport {
		score -= cfg.RestFallbackPenalty
	}
	h.Score = ClampScore(score)
	switch {
	case h.Score >= 90 && h.DataFresh:
		h.Status = "ok"
	case h.ProviderType == "broker" && h.Enabled && !h.GatewayConnected:
		h.Status = "disconnected"
	case h.WebSocketEnabled && !h.WebSocketConnected && !h.RestFallbackActive:
		h.Status = "disconnected"
	case !h.DataFresh:
		h.Status = "stale"
	default:
		h.Status = "degraded"
	}
	return h
}

func ClampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
