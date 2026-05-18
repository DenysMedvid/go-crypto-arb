package config

import (
	"fmt"
	"sort"
	"strings"

	"go-crypto-arb/internal/exchange"
)

type ValidationLevel string

const (
	ValidationOK    ValidationLevel = "OK"
	ValidationWarn  ValidationLevel = "WARN"
	ValidationError ValidationLevel = "ERROR"
)

type ValidationMessage struct {
	Level   ValidationLevel `json:"level"`
	Message string          `json:"message"`
}

func Validate(cfg Config, env Env, markets []exchange.MarketInfo) []ValidationMessage {
	var out []ValidationMessage
	if env.APIKey == "" {
		out = append(out, ValidationMessage{Level: ValidationError, Message: "API_KEY is missing"})
	} else {
		out = append(out, ValidationMessage{Level: ValidationOK, Message: "API_KEY is configured"})
	}
	for name, ex := range cfg.Exchanges {
		if !ex.Enabled {
			continue
		}
		if exchange.IsSupportedCryptoPlatform(name) {
			out = append(out, ValidationMessage{Level: ValidationOK, Message: fmt.Sprintf("%s exchange is supported", titleExchange(name))})
		} else {
			out = append(out, ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("%s exchange is not supported", name)})
		}
		if ex.Fees.SpotTaker.DecimalValue().IsZero() {
			out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s spot taker fee is zero or missing", titleExchange(name))})
		}
		if ex.FuturesEnabled && ex.Fees.FuturesTaker.DecimalValue().IsZero() {
			out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s futures taker fee is zero or missing", titleExchange(name))})
		}
		if strings.EqualFold(name, "kraken") && ex.FuturesEnabled {
			out = append(out, ValidationMessage{Level: ValidationWarn, Message: "Kraken futures support is partial in v2"})
		}
		if isSpotOnlyPublicRESTPlatform(name) && ex.FuturesEnabled {
			out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s futures support is not implemented; spot market data only", titleExchange(name))})
		}
		if isSpotOnlyPublicRESTPlatform(name) && ex.WebsocketEnabled {
			out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s websocket support is not implemented; REST polling is used", titleExchange(name))})
		}
	}
	for name, provider := range cfg.Providers {
		if !provider.Enabled {
			continue
		}
		switch strings.ToLower(provider.Type) {
		case "crypto_exchange":
			if !exchange.IsSupportedCryptoPlatform(name) {
				out = append(out, ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("%s crypto exchange provider is not supported", name)})
			} else {
				out = append(out, ValidationMessage{Level: ValidationOK, Message: fmt.Sprintf("%s crypto exchange provider is supported", titleExchange(name))})
				if isSpotOnlyPublicRESTPlatform(name) && provider.FuturesEnabled {
					out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s futures support is not implemented; spot market data only", titleExchange(name))})
				}
				if isSpotOnlyPublicRESTPlatform(name) && provider.WebsocketEnabled {
					out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s websocket support is not implemented; REST polling is used", titleExchange(name))})
				}
			}
		case "broker":
			if !strings.EqualFold(name, "ibkr") {
				out = append(out, ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("%s broker provider is not supported", name)})
				continue
			}
			out = append(out, validateIBKRProvider(provider, cfg.InstrumentUniverses)...)
		default:
			out = append(out, ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("%s provider type %q is not supported", name, provider.Type)})
		}
	}
	out = append(out, validateStrategyProviders(cfg)...)
	if len(cfg.Assets.Preferred) == 0 {
		out = append(out, ValidationMessage{Level: ValidationError, Message: "assets.preferred is empty"})
	}
	for _, stable := range cfg.Assets.StableBases {
		stable = NormalizeAssetName(stable)
		if !containsAsset(cfg.Assets.Preferred, stable) {
			out = append(out, ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("stable base %s is not present in assets.preferred", stable)})
		} else {
			out = append(out, ValidationMessage{Level: ValidationOK, Message: fmt.Sprintf("stable base %s is configured", stable)})
		}
	}
	for _, quote := range cfg.Assets.QuoteAssets {
		quote = NormalizeAssetName(quote)
		if !containsAsset(cfg.Assets.Preferred, quote) {
			out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("quote asset %s is not present in assets.preferred", quote)})
		}
	}
	if len(markets) > 0 {
		out = append(out, validateMarkets(cfg, markets)...)
	} else {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "market availability checks skipped because no discovered markets were provided"})
	}
	return out
}

func validateStrategyProviders(cfg Config) []ValidationMessage {
	known := make(map[string]struct{})
	for name := range cfg.Providers {
		known[strings.ToLower(name)] = struct{}{}
	}
	for name := range cfg.Exchanges {
		known[strings.ToLower(name)] = struct{}{}
	}
	check := func(strategy string, providers []string) []ValidationMessage {
		var messages []ValidationMessage
		for _, provider := range providers {
			if _, ok := known[strings.ToLower(provider)]; !ok {
				messages = append(messages, ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("%s references unknown provider %s", strategy, provider)})
			}
		}
		return messages
	}
	var out []ValidationMessage
	out = append(out, check("crypto_triangular", cfg.Strategies.CryptoTriangular.Providers)...)
	out = append(out, check("crypto_triangular exclude_providers", cfg.Strategies.CryptoTriangular.ExcludeProviders)...)
	out = append(out, check("cross_exchange", cfg.Strategies.CrossExchange.Providers)...)
	out = append(out, check("crypto_spot_futures spot_providers", cfg.Strategies.CryptoSpotFutures.SpotProviders)...)
	out = append(out, check("crypto_spot_futures futures_providers", cfg.Strategies.CryptoSpotFutures.FuturesProviders)...)
	out = append(out, check("ibkr_fx_triangular", cfg.Strategies.IBKRFXTriangular.Providers)...)
	out = append(out, check("crypto_spot_vs_ibkr_futures spot_providers", cfg.Strategies.CryptoSpotVsIBKRFutures.SpotProviders)...)
	out = append(out, check("crypto_spot_vs_ibkr_futures futures_providers", cfg.Strategies.CryptoSpotVsIBKRFutures.FuturesProviders)...)
	return out
}

func validateIBKRProvider(provider ProviderConfig, universes map[string]InstrumentUniverse) []ValidationMessage {
	var out []ValidationMessage
	if provider.Host == "" {
		out = append(out, ValidationMessage{Level: ValidationError, Message: "IBKR host is missing"})
	}
	if provider.Port == 0 {
		out = append(out, ValidationMessage{Level: ValidationError, Message: "IBKR port is missing"})
	}
	if provider.ClientID == 0 {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "IBKR client_id is missing; default 101 will be used if loaded from config"})
	}
	if provider.TradingEnabled {
		out = append(out, ValidationMessage{Level: ValidationError, Message: "IBKR trading_enabled must remain false in v2.1"})
	} else {
		out = append(out, ValidationMessage{Level: ValidationOK, Message: "IBKR trading is disabled"})
	}
	if provider.CryptoSpotEnabled {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "IBKR crypto spot is explicitly enabled; this is disabled by default and not used for crypto strategies unless configured"})
	} else {
		out = append(out, ValidationMessage{Level: ValidationOK, Message: "IBKR crypto spot is disabled"})
	}
	seen := 0
	for _, universe := range universes {
		if !providerListContains(universe.Providers, "ibkr") {
			continue
		}
		for _, item := range universe.Instruments {
			seen++
			out = append(out, validateIBKRInstrument(item))
		}
	}
	if seen == 0 {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "IBKR is enabled but no IBKR instruments are configured"})
	}
	if provider.Fees.FXEstimatedTaker.DecimalValue().IsZero() && provider.FXEnabled {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "IBKR FX estimated taker fee is zero or missing"})
	}
	if provider.Fees.FuturesEstimatedTaker.DecimalValue().IsZero() && provider.FuturesEnabled {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "IBKR futures estimated taker fee is zero or missing"})
	}
	return out
}

func validateIBKRInstrument(item InstrumentConfig) ValidationMessage {
	if item.ID == "" {
		return ValidationMessage{Level: ValidationError, Message: "IBKR instrument id is missing"}
	}
	if item.Symbol == "" && item.IBKR.Symbol == "" {
		return ValidationMessage{Level: ValidationError, Message: fmt.Sprintf("IBKR instrument %s symbol is missing", item.ID)}
	}
	if item.IBKR.Symbol != "" {
		if item.IBKR.SecType == "" || item.IBKR.Exchange == "" || item.IBKR.Currency == "" {
			return ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("IBKR instrument %s is missing sec_type/exchange/currency fields", item.ID)}
		}
	}
	return ValidationMessage{Level: ValidationOK, Message: fmt.Sprintf("IBKR instrument %s is configured", item.ID)}
}

func HasValidationErrors(messages []ValidationMessage) bool {
	for _, message := range messages {
		if message.Level == ValidationError {
			return true
		}
	}
	return false
}

func validateMarkets(cfg Config, markets []exchange.MarketInfo) []ValidationMessage {
	var out []ValidationMessage
	marketSet := make(map[string]exchange.MarketInfo)
	enabledExchanges := make([]string, 0, len(cfg.Exchanges))
	for name, ex := range cfg.Exchanges {
		if ex.Enabled {
			enabledExchanges = append(enabledExchanges, titleExchange(name))
		}
	}
	sort.Strings(enabledExchanges)
	for _, market := range markets {
		if market.Active {
			marketSet[marketKey(market.Exchange, market.MarketType, market.Symbol)] = market
		}
	}
	for _, exchangeName := range enabledExchanges {
		exCfg, _ := exchangeConfigByName(cfg, exchangeName)
		for _, base := range cfg.Assets.Preferred {
			base = NormalizeAssetName(base)
			for _, quote := range cfg.Assets.QuoteAssets {
				quote = NormalizeAssetName(quote)
				if base == quote {
					continue
				}
				symbol := exchange.CanonicalSymbol(base, quote)
				if _, ok := marketSet[marketKey(exchangeName, exchange.MarketSpot, symbol)]; ok {
					out = append(out, ValidationMessage{Level: ValidationOK, Message: fmt.Sprintf("%s spot %s is available", exchangeName, symbol)})
				}
				if exCfg.FuturesEnabled {
					if _, ok := marketSet[marketKey(exchangeName, exchange.MarketFutures, symbol)]; !ok {
						out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s futures %s is unavailable", exchangeName, symbol)})
					}
				}
			}
		}
	}
	if !triangularPossible(cfg, marketSet) {
		out = append(out, ValidationMessage{Level: ValidationWarn, Message: "no triangular cycles detected in discovered markets"})
	}
	out = append(out, crossExchangeMarketWarnings(enabledExchanges, markets)...)
	return out
}

func crossExchangeMarketWarnings(enabledExchanges []string, markets []exchange.MarketInfo) []ValidationMessage {
	if len(enabledExchanges) < 2 {
		return nil
	}
	exchangeEnabled := make(map[string]struct{})
	for _, name := range enabledExchanges {
		exchangeEnabled[strings.ToLower(name)] = struct{}{}
	}
	bySymbol := make(map[string]map[string]struct{})
	for _, market := range markets {
		if market.MarketType != exchange.MarketSpot || !market.Active {
			continue
		}
		if _, ok := exchangeEnabled[strings.ToLower(market.Exchange)]; !ok {
			continue
		}
		if _, ok := bySymbol[market.Symbol]; !ok {
			bySymbol[market.Symbol] = make(map[string]struct{})
		}
		bySymbol[market.Symbol][titleExchange(market.Exchange)] = struct{}{}
	}
	var out []ValidationMessage
	for symbol, exchanges := range bySymbol {
		if len(exchanges) == len(enabledExchanges) {
			continue
		}
		for _, name := range enabledExchanges {
			if _, ok := exchanges[name]; !ok {
				out = append(out, ValidationMessage{Level: ValidationWarn, Message: fmt.Sprintf("%s exists on another exchange but not on %s", symbol, name)})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Message < out[j].Message })
	return out
}

func exchangeConfigByName(cfg Config, name string) (ExchangeConfig, bool) {
	if ex, ok := cfg.Exchanges[name]; ok {
		return ex, true
	}
	for configuredName, ex := range cfg.Exchanges {
		if strings.EqualFold(configuredName, name) {
			return ex, true
		}
	}
	return ExchangeConfig{}, false
}

func triangularPossible(cfg Config, marketSet map[string]exchange.MarketInfo) bool {
	for name, ex := range cfg.Exchanges {
		if !ex.Enabled {
			continue
		}
		exchangeName := titleExchange(name)
		for _, stable := range cfg.Assets.StableBases {
			for _, a := range cfg.Assets.Preferred {
				for _, b := range cfg.Assets.Preferred {
					stable = NormalizeAssetName(stable)
					a = NormalizeAssetName(a)
					b = NormalizeAssetName(b)
					if stable == a || stable == b || a == b {
						continue
					}
					if pairExists(marketSet, exchangeName, stable, a) && pairExists(marketSet, exchangeName, a, b) && pairExists(marketSet, exchangeName, b, stable) {
						return true
					}
				}
			}
		}
	}
	return false
}

func pairExists(markets map[string]exchange.MarketInfo, exchangeName, a, b string) bool {
	_, direct := markets[marketKey(exchangeName, exchange.MarketSpot, exchange.CanonicalSymbol(a, b))]
	_, inverse := markets[marketKey(exchangeName, exchange.MarketSpot, exchange.CanonicalSymbol(b, a))]
	return direct || inverse
}

func marketKey(exchangeName string, marketType exchange.MarketType, symbol string) string {
	return strings.ToLower(exchangeName) + "|" + string(marketType) + "|" + exchange.NormalizeCanonicalSymbol(symbol)
}

func containsAsset(assets []string, target string) bool {
	target = NormalizeAssetName(target)
	for _, asset := range assets {
		if NormalizeAssetName(asset) == target {
			return true
		}
	}
	return false
}

func providerListContains(providers []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, provider := range providers {
		if strings.ToLower(strings.TrimSpace(provider)) == target {
			return true
		}
	}
	return false
}

func titleExchange(name string) string {
	return exchange.DisplayName(name)
}

func isSpotOnlyPublicRESTPlatform(name string) bool {
	switch strings.ToLower(name) {
	case "okx", "bybit", "coinbase", "gateio", "bitget":
		return true
	default:
		return false
	}
}
