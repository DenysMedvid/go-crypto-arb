package alerts

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

type AlertType string

const (
	AlertTriangular       AlertType = "triangular_arbitrage"
	AlertCrossExchange    AlertType = "cross_exchange_arbitrage"
	AlertSpotFutures      AlertType = "spot_futures_basis"
	AlertIBKRFXTriangular AlertType = "ibkr_fx_triangular_arbitrage"
	AlertBrokerBasis      AlertType = "crypto_spot_vs_ibkr_futures_basis"
	AlertExchangeStale    AlertType = "exchange_data_stale"
	AlertExchangeOffline  AlertType = "exchange_disconnected"
	AlertHealthScoreLow   AlertType = "health_score_low"
)

type Alert struct {
	ID          string          `json:"id"`
	DedupKey    string          `json:"dedup_key"`
	Severity    AlertSeverity   `json:"severity"`
	Type        AlertType       `json:"type"`
	Message     string          `json:"message"`
	Exchange    string          `json:"exchange,omitempty"`
	Symbol      string          `json:"symbol,omitempty"`
	Value       decimal.Decimal `json:"value"`
	Threshold   decimal.Decimal `json:"threshold"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	RepeatCount int             `json:"repeat_count"`
	Status      string          `json:"status"`
}

// TODO: Add Telegram, email, and webhook implementations behind this interface.
type Notifier interface {
	Notify(ctx context.Context, alert Alert) error
}

type Engine struct {
	latest map[string]Alert
}

func NewEngine() *Engine {
	return &Engine{latest: make(map[string]Alert)}
}

func (e *Engine) Evaluate(cfg config.Config, tri []arbitrage.TriangularOpportunityV2, cross []arbitrage.CrossExchangeOpportunityV2, spotFutures []arbitrage.SpotFuturesOpportunityV2, ibkrFX []arbitrage.TriangularOpportunityV2, brokerBasis []arbitrage.BrokerFuturesBasisOpportunity, health []exchange.ExchangeHealth) []Alert {
	if !cfg.Alerts.Enabled {
		return nil
	}
	now := time.Now()
	profitThreshold := cfg.Alerts.MinProfitPercent.DecimalValue()
	basisThreshold := cfg.Alerts.MinBasisPercent.DecimalValue()

	for _, item := range tri {
		if item.NetProfitPercent.LessThan(profitThreshold) {
			continue
		}
		key := strings.Join([]string{string(AlertTriangular), item.Exchange, strings.Join(item.Cycle, ">"), item.StartAmount.String()}, "|")
		e.upsert(cfg, Alert{
			DedupKey:  key,
			Type:      AlertTriangular,
			Severity:  severityForProfit(item.NetProfitPercent, profitThreshold),
			Message:   fmt.Sprintf("%s triangular %s size %s", item.Exchange, item.NetProfitPercent.StringFixed(3)+"%", item.StartAmount.String()),
			Exchange:  item.Exchange,
			Value:     item.NetProfitPercent,
			Threshold: profitThreshold,
			Status:    "active",
		}, now)
	}
	for _, item := range cross {
		if item.NetProfitPercent.LessThan(profitThreshold) {
			continue
		}
		key := strings.Join([]string{string(AlertCrossExchange), item.Symbol, item.BuyExchange, item.SellExchange, item.TradeSize.String()}, "|")
		e.upsert(cfg, Alert{
			DedupKey:  key,
			Type:      AlertCrossExchange,
			Severity:  severityForProfit(item.NetProfitPercent, profitThreshold),
			Message:   fmt.Sprintf("%s cross-exchange %s size %s", item.Symbol, item.NetProfitPercent.StringFixed(3)+"%", item.TradeSize.String()),
			Symbol:    item.Symbol,
			Value:     item.NetProfitPercent,
			Threshold: profitThreshold,
			Status:    "active",
		}, now)
	}
	for _, item := range spotFutures {
		if item.BasisPercent.LessThan(basisThreshold) {
			continue
		}
		key := strings.Join([]string{string(AlertSpotFutures), item.Exchange, item.Symbol, item.TradeSize.String()}, "|")
		e.upsert(cfg, Alert{
			DedupKey:  key,
			Type:      AlertSpotFutures,
			Severity:  severityForProfit(item.BasisPercent, basisThreshold),
			Message:   fmt.Sprintf("%s %s spot-futures basis %s size %s", item.Exchange, item.Symbol, item.BasisPercent.StringFixed(3)+"%", item.TradeSize.String()),
			Exchange:  item.Exchange,
			Symbol:    item.Symbol,
			Value:     item.BasisPercent,
			Threshold: basisThreshold,
			Status:    "active",
		}, now)
	}
	for _, item := range ibkrFX {
		if item.NetProfitPercent.LessThan(profitThreshold) {
			continue
		}
		key := strings.Join([]string{string(AlertIBKRFXTriangular), item.Provider, strings.Join(item.Cycle, ">"), item.StartAmount.String()}, "|")
		e.upsert(cfg, Alert{
			DedupKey:  key,
			Type:      AlertIBKRFXTriangular,
			Severity:  severityForProfit(item.NetProfitPercent, profitThreshold),
			Message:   fmt.Sprintf("IBKR FX triangular %s size %s", item.NetProfitPercent.StringFixed(3)+"%", item.StartAmount.String()),
			Exchange:  item.Exchange,
			Value:     item.NetProfitPercent,
			Threshold: profitThreshold,
			Status:    "active",
		}, now)
	}
	for _, item := range brokerBasis {
		if item.BasisPercent.LessThan(basisThreshold) {
			continue
		}
		key := strings.Join([]string{string(AlertBrokerBasis), item.Asset, item.SpotProvider, item.FuturesInstrumentID}, "|")
		e.upsert(cfg, Alert{
			DedupKey:  key,
			Type:      AlertBrokerBasis,
			Severity:  severityForProfit(item.BasisPercent, basisThreshold),
			Message:   fmt.Sprintf("%s spot vs IBKR futures basis %s", item.Asset, item.BasisPercent.StringFixed(3)+"%"),
			Exchange:  item.FuturesProvider,
			Symbol:    item.Asset,
			Value:     item.BasisPercent,
			Threshold: basisThreshold,
			Status:    "active",
		}, now)
	}
	for _, item := range health {
		if !item.DataFresh || item.StaleTickerCount > 0 || item.StaleOrderBookCount > 0 {
			e.upsert(cfg, Alert{
				DedupKey:  string(AlertExchangeStale) + "|" + item.Exchange,
				Type:      AlertExchangeStale,
				Severity:  AlertWarning,
				Message:   fmt.Sprintf("%s market data stale", item.Exchange),
				Exchange:  item.Exchange,
				Value:     decimal.NewFromInt(int64(item.StaleTickerCount + item.StaleOrderBookCount)),
				Threshold: decimal.Zero,
				Status:    "active",
			}, now)
		}
		if item.WebSocketEnabled && !item.WebSocketConnected && !item.RestFallbackActive {
			e.upsert(cfg, Alert{
				DedupKey:  string(AlertExchangeOffline) + "|" + item.Exchange,
				Type:      AlertExchangeOffline,
				Severity:  AlertCritical,
				Message:   fmt.Sprintf("%s disconnected", item.Exchange),
				Exchange:  item.Exchange,
				Value:     decimal.Zero,
				Threshold: decimal.Zero,
				Status:    "active",
			}, now)
		}
		if item.Score > 0 && item.Score < 60 {
			e.upsert(cfg, Alert{
				DedupKey:  string(AlertHealthScoreLow) + "|" + item.Exchange,
				Type:      AlertHealthScoreLow,
				Severity:  AlertCritical,
				Message:   fmt.Sprintf("%s health score low: %d", item.Exchange, item.Score),
				Exchange:  item.Exchange,
				Value:     decimal.NewFromInt(int64(item.Score)),
				Threshold: decimal.NewFromInt(60),
				Status:    "active",
			}, now)
		}
	}

	out := make([]Alert, 0, len(e.latest))
	for _, alert := range e.latest {
		out = append(out, alert)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity == out[j].Severity {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	if cfg.Alerts.MaxResults > 0 && len(out) > cfg.Alerts.MaxResults {
		return out[:cfg.Alerts.MaxResults]
	}
	return out
}

func (e *Engine) upsert(cfg config.Config, next Alert, now time.Time) {
	next.ID = stableAlertID(next.DedupKey)
	next.CreatedAt = now
	next.UpdatedAt = now
	previous, ok := e.latest[next.DedupKey]
	if !ok {
		e.latest[next.DedupKey] = next
		return
	}
	cooldownActive := now.Sub(previous.UpdatedAt) < cfg.Alerts.Cooldown.Duration
	changeThreshold := cfg.Alerts.RepeatIfProfitChangesByPercent.DecimalValue()
	change := next.Value.Sub(previous.Value).Abs()
	if cooldownActive && (changeThreshold.IsZero() || change.LessThan(changeThreshold)) {
		return
	}
	next.ID = previous.ID
	next.CreatedAt = previous.CreatedAt
	next.RepeatCount = previous.RepeatCount + 1
	e.latest[next.DedupKey] = next
}

func stableAlertID(key string) string {
	return strings.NewReplacer("|", "-", "/", "_", ">", "_").Replace(strings.ToLower(key))
}

func severityForProfit(value, threshold decimal.Decimal) AlertSeverity {
	if threshold.IsZero() {
		if value.GreaterThan(decimal.Zero) {
			return AlertInfo
		}
		return AlertWarning
	}
	if value.GreaterThanOrEqual(threshold.Mul(decimal.NewFromInt(2))) {
		return AlertWarning
	}
	return AlertInfo
}

func severityRank(severity AlertSeverity) int {
	switch severity {
	case AlertCritical:
		return 3
	case AlertWarning:
		return 2
	default:
		return 1
	}
}
