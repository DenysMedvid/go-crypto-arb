package instrument

import (
	"strings"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

const (
	AssetClassCrypto  = "crypto"
	AssetClassFX      = "fx"
	AssetClassFutures = "futures"
	AssetClassStock   = "stock"
	AssetClassETF     = "etf"
)

func IBKRInstruments(cfg config.Config) []config.InstrumentConfig {
	var out []config.InstrumentConfig
	for _, universe := range cfg.InstrumentUniverses {
		if !contains(universe.Providers, "ibkr") {
			continue
		}
		for _, item := range universe.Instruments {
			if len(item.Providers) > 0 && !contains(item.Providers, "ibkr") {
				continue
			}
			out = append(out, item)
		}
	}
	return out
}

func UniverseInstruments(cfg config.Config, universeName string) []config.InstrumentConfig {
	universe, ok := cfg.InstrumentUniverses[universeName]
	if !ok {
		return nil
	}
	return append([]config.InstrumentConfig(nil), universe.Instruments...)
}

func MarketType(value string) exchange.MarketType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(exchange.MarketFutures), "future", "fut":
		return exchange.MarketFutures
	default:
		return exchange.MarketSpot
	}
}

func DisplaySymbol(item config.InstrumentConfig) string {
	if item.Symbol != "" {
		return exchange.NormalizeCanonicalSymbol(item.Symbol)
	}
	if item.IBKR.Symbol != "" {
		return item.IBKR.Symbol
	}
	return item.ID
}

func contains(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}
