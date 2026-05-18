package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", raw, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", d.String())), nil
}

type Env struct {
	APIKey       string
	ConfigPath   string
	HTTPAddr     string
	IBKREnabled  string
	IBKRHost     string
	IBKRPort     string
	IBKRClientID string
}

type Config struct {
	App                 AppConfig                     `yaml:"app" json:"app"`
	API                 APIConfig                     `yaml:"api" json:"api"`
	Providers           map[string]ProviderConfig     `yaml:"providers" json:"providers"`
	Exchanges           map[string]ExchangeConfig     `yaml:"exchanges" json:"exchanges"`
	MarketData          MarketDataConfig              `yaml:"market_data" json:"market_data"`
	Simulation          SimulationConfig              `yaml:"simulation" json:"simulation"`
	Assets              AssetConfig                   `yaml:"assets" json:"assets"`
	InstrumentUniverses map[string]InstrumentUniverse `yaml:"instrument_universes" json:"instrument_universes"`
	Strategies          StrategiesConfig              `yaml:"strategies" json:"strategies"`
	Arbitrage           ArbitrageConfig               `yaml:"arbitrage" json:"arbitrage"`
	Signals             SignalsConfig                 `yaml:"signals" json:"signals"`
	Alerts              AlertsConfig                  `yaml:"alerts" json:"alerts"`
	Metrics             MetricsConfig                 `yaml:"metrics" json:"metrics"`
	Health              HealthConfig                  `yaml:"health" json:"health"`
	TUI                 TUIConfig                     `yaml:"tui" json:"tui"`
}

type AppConfig struct {
	Version         string   `yaml:"version" json:"version"`
	RefreshInterval Duration `yaml:"refresh_interval" json:"refresh_interval"`
	StalePriceAfter Duration `yaml:"stale_price_after" json:"stale_price_after"`
}

type APIConfig struct {
	HTTPAddr           string   `yaml:"http_addr" json:"http_addr"`
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins" json:"cors_allowed_origins"`
}

type ExchangeConfig struct {
	Enabled          bool         `yaml:"enabled" json:"enabled"`
	SpotEnabled      bool         `yaml:"spot_enabled" json:"spot_enabled"`
	FuturesEnabled   bool         `yaml:"futures_enabled" json:"futures_enabled"`
	WebsocketEnabled bool         `yaml:"websocket_enabled" json:"websocket_enabled"`
	RestPollInterval Duration     `yaml:"rest_poll_interval" json:"rest_poll_interval"`
	Fees             ExchangeFees `yaml:"fees" json:"fees"`
}

type ExchangeFees struct {
	SpotTaker    Decimal `yaml:"spot_taker" json:"spot_taker"`
	FuturesTaker Decimal `yaml:"futures_taker" json:"futures_taker"`
}

type ProviderConfig struct {
	Enabled           bool         `yaml:"enabled" json:"enabled"`
	Type              string       `yaml:"type" json:"type"`
	SpotEnabled       bool         `yaml:"spot_enabled" json:"spot_enabled"`
	FuturesEnabled    bool         `yaml:"futures_enabled" json:"futures_enabled"`
	WebsocketEnabled  bool         `yaml:"websocket_enabled" json:"websocket_enabled"`
	RestPollInterval  Duration     `yaml:"rest_poll_interval" json:"rest_poll_interval"`
	MarketDataEnabled bool         `yaml:"market_data_enabled" json:"market_data_enabled"`
	TradingEnabled    bool         `yaml:"trading_enabled" json:"trading_enabled"`
	CryptoSpotEnabled bool         `yaml:"crypto_spot_enabled" json:"crypto_spot_enabled"`
	FXEnabled         bool         `yaml:"fx_enabled" json:"fx_enabled"`
	StocksEnabled     bool         `yaml:"stocks_enabled" json:"stocks_enabled"`
	ETFsEnabled       bool         `yaml:"etfs_enabled" json:"etfs_enabled"`
	APIMode           string       `yaml:"api_mode" json:"api_mode"`
	Host              string       `yaml:"host" json:"host"`
	Port              int          `yaml:"port" json:"port"`
	ClientID          int          `yaml:"client_id" json:"client_id"`
	Fees              ProviderFees `yaml:"fees" json:"fees"`
}

type ProviderFees struct {
	SpotTaker             Decimal `yaml:"spot_taker" json:"spot_taker"`
	FuturesTaker          Decimal `yaml:"futures_taker" json:"futures_taker"`
	FXEstimatedTaker      Decimal `yaml:"fx_estimated_taker" json:"fx_estimated_taker"`
	FuturesEstimatedTaker Decimal `yaml:"futures_estimated_taker" json:"futures_estimated_taker"`
}

type MarketDataConfig struct {
	OrderBookDepth      int      `yaml:"order_book_depth" json:"order_book_depth"`
	OrderBookStaleAfter Duration `yaml:"order_book_stale_after" json:"order_book_stale_after"`
	TickerStaleAfter    Duration `yaml:"ticker_stale_after" json:"ticker_stale_after"`
}

type SimulationConfig struct {
	Enabled            bool      `yaml:"enabled" json:"enabled"`
	QuoteAsset         string    `yaml:"quote_asset" json:"quote_asset"`
	TradeSizes         []Decimal `yaml:"trade_sizes" json:"trade_sizes"`
	MaxSlippagePercent Decimal   `yaml:"max_slippage_percent" json:"max_slippage_percent"`
}

type AssetConfig struct {
	Preferred   []string `yaml:"preferred" json:"preferred"`
	StableBases []string `yaml:"stable_bases" json:"stable_bases"`
	QuoteAssets []string `yaml:"quote_assets" json:"quote_assets"`
}

type InstrumentUniverse struct {
	Title       string             `yaml:"title" json:"title"`
	Providers   []string           `yaml:"providers" json:"providers"`
	Instruments []InstrumentConfig `yaml:"instruments" json:"instruments"`
}

type InstrumentConfig struct {
	ID          string               `yaml:"id" json:"id"`
	DisplayName string               `yaml:"display_name" json:"display_name"`
	Symbol      string               `yaml:"symbol" json:"symbol"`
	AssetClass  string               `yaml:"asset_class" json:"asset_class"`
	MarketType  string               `yaml:"market_type" json:"market_type"`
	Providers   []string             `yaml:"providers" json:"providers"`
	IBKR        IBKRInstrumentConfig `yaml:"ibkr" json:"ibkr"`
}

type IBKRInstrumentConfig struct {
	Symbol   string `yaml:"symbol" json:"symbol"`
	SecType  string `yaml:"sec_type" json:"sec_type"`
	Exchange string `yaml:"exchange" json:"exchange"`
	Currency string `yaml:"currency" json:"currency"`
	ConID    *int64 `yaml:"con_id" json:"con_id"`
}

type ArbitrageConfig struct {
	Triangular    ResultLimitConfig    `yaml:"triangular" json:"triangular"`
	CrossExchange ResultLimitConfig    `yaml:"cross_exchange" json:"cross_exchange"`
	SpotFutures   SpotFuturesArbConfig `yaml:"spot_futures" json:"spot_futures"`
}

type StrategiesConfig struct {
	CryptoTriangular        StrategyConfig             `yaml:"crypto_triangular" json:"crypto_triangular"`
	CrossExchange           StrategyConfig             `yaml:"cross_exchange" json:"cross_exchange"`
	CryptoSpotFutures       SpotFuturesStrategyConfig  `yaml:"crypto_spot_futures" json:"crypto_spot_futures"`
	IBKRFXTriangular        FXTriangularStrategyConfig `yaml:"ibkr_fx_triangular" json:"ibkr_fx_triangular"`
	CryptoSpotVsIBKRFutures BrokerFuturesBasisStrategy `yaml:"crypto_spot_vs_ibkr_futures" json:"crypto_spot_vs_ibkr_futures"`
}

type StrategyConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	Title             string   `yaml:"title" json:"title"`
	Providers         []string `yaml:"providers" json:"providers"`
	Universe          string   `yaml:"universe" json:"universe"`
	BaseAssets        []string `yaml:"base_assets" json:"base_assets"`
	ExcludeProviders  []string `yaml:"exclude_providers" json:"exclude_providers"`
	MinProfitPercent  Decimal  `yaml:"min_profit_percent" json:"min_profit_percent"`
	MaxResults        int      `yaml:"max_results" json:"max_results"`
	UseOrderBookDepth bool     `yaml:"use_order_book_depth" json:"use_order_book_depth"`
}

type SpotFuturesStrategyConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	Title              string   `yaml:"title" json:"title"`
	SpotProviders      []string `yaml:"spot_providers" json:"spot_providers"`
	FuturesProviders   []string `yaml:"futures_providers" json:"futures_providers"`
	MinBasisPercent    Decimal  `yaml:"min_basis_percent" json:"min_basis_percent"`
	IncludeFundingRate bool     `yaml:"include_funding_rate" json:"include_funding_rate"`
	MaxResults         int      `yaml:"max_results" json:"max_results"`
	UseOrderBookDepth  bool     `yaml:"use_order_book_depth" json:"use_order_book_depth"`
}

type FXTriangularStrategyConfig struct {
	Enabled           bool       `yaml:"enabled" json:"enabled"`
	Title             string     `yaml:"title" json:"title"`
	Providers         []string   `yaml:"providers" json:"providers"`
	Universe          string     `yaml:"universe" json:"universe"`
	BaseAssets        []string   `yaml:"base_assets" json:"base_assets"`
	Cycles            [][]string `yaml:"cycles" json:"cycles"`
	MinProfitPercent  Decimal    `yaml:"min_profit_percent" json:"min_profit_percent"`
	MaxResults        int        `yaml:"max_results" json:"max_results"`
	UseOrderBookDepth bool       `yaml:"use_order_book_depth" json:"use_order_book_depth"`
}

type BrokerFuturesBasisStrategy struct {
	Enabled           bool                           `yaml:"enabled" json:"enabled"`
	Title             string                         `yaml:"title" json:"title"`
	SpotProviders     []string                       `yaml:"spot_providers" json:"spot_providers"`
	FuturesProviders  []string                       `yaml:"futures_providers" json:"futures_providers"`
	CryptoSpotViaIBKR bool                           `yaml:"crypto_spot_via_ibkr" json:"crypto_spot_via_ibkr"`
	MinBasisPercent   Decimal                        `yaml:"min_basis_percent" json:"min_basis_percent"`
	MaxResults        int                            `yaml:"max_results" json:"max_results"`
	UseOrderBookDepth bool                           `yaml:"use_order_book_depth" json:"use_order_book_depth"`
	Instruments       []BrokerFuturesBasisInstrument `yaml:"instruments" json:"instruments"`
}

type BrokerFuturesBasisInstrument struct {
	ID            string                   `yaml:"id" json:"id"`
	Asset         string                   `yaml:"asset" json:"asset"`
	SpotSymbols   []ProviderSymbol         `yaml:"spot_symbols" json:"spot_symbols"`
	FuturesSymbol ProviderInstrumentSymbol `yaml:"futures_symbol" json:"futures_symbol"`
}

type ProviderSymbol struct {
	Provider string `yaml:"provider" json:"provider"`
	Symbol   string `yaml:"symbol" json:"symbol"`
}

type ProviderInstrumentSymbol struct {
	Provider     string `yaml:"provider" json:"provider"`
	InstrumentID string `yaml:"instrument_id" json:"instrument_id"`
}

type ResultLimitConfig struct {
	Enabled           bool    `yaml:"enabled" json:"enabled"`
	MinProfitPercent  Decimal `yaml:"min_profit_percent" json:"min_profit_percent"`
	MaxResults        int     `yaml:"max_results" json:"max_results"`
	UseOrderBookDepth bool    `yaml:"use_order_book_depth" json:"use_order_book_depth"`
}

type SpotFuturesArbConfig struct {
	Enabled            bool    `yaml:"enabled" json:"enabled"`
	MinBasisPercent    Decimal `yaml:"min_basis_percent" json:"min_basis_percent"`
	IncludeFundingRate bool    `yaml:"include_funding_rate" json:"include_funding_rate"`
	MaxResults         int     `yaml:"max_results" json:"max_results"`
	UseOrderBookDepth  bool    `yaml:"use_order_book_depth" json:"use_order_book_depth"`
}

type SignalsConfig struct {
	RelatedAssets RelatedAssetsConfig `yaml:"related_assets" json:"related_assets"`
}

type RelatedAssetsConfig struct {
	Enabled    bool         `yaml:"enabled" json:"enabled"`
	Title      string       `yaml:"title" json:"title"`
	MaxResults int          `yaml:"max_results" json:"max_results"`
	Groups     []AssetGroup `yaml:"groups" json:"groups"`
}

type AssetGroup struct {
	Name   string   `yaml:"name" json:"name"`
	Assets []string `yaml:"assets" json:"assets"`
}

type AlertsConfig struct {
	Enabled                        bool     `yaml:"enabled" json:"enabled"`
	Title                          string   `yaml:"title" json:"title"`
	MinProfitPercent               Decimal  `yaml:"min_profit_percent" json:"min_profit_percent"`
	MinBasisPercent                Decimal  `yaml:"min_basis_percent" json:"min_basis_percent"`
	Cooldown                       Duration `yaml:"cooldown" json:"cooldown"`
	RepeatIfProfitChangesByPercent Decimal  `yaml:"repeat_if_profit_changes_by_percent" json:"repeat_if_profit_changes_by_percent"`
	MaxResults                     int      `yaml:"max_results" json:"max_results"`
}

type MetricsConfig struct {
	PrometheusEnabled bool   `yaml:"prometheus_enabled" json:"prometheus_enabled"`
	PrometheusPath    string `yaml:"prometheus_path" json:"prometheus_path"`
}

type HealthConfig struct {
	ScoringEnabled      bool `yaml:"scoring_enabled" json:"scoring_enabled"`
	StalePenalty        int  `yaml:"stale_penalty" json:"stale_penalty"`
	DisconnectedPenalty int  `yaml:"disconnected_penalty" json:"disconnected_penalty"`
	RestFallbackPenalty int  `yaml:"rest_fallback_penalty" json:"rest_fallback_penalty"`
	ReconnectPenalty    int  `yaml:"reconnect_penalty" json:"reconnect_penalty"`
}

type TUIConfig struct {
	BackendURL       string                  `yaml:"backend_url" json:"backend_url"`
	RefreshInterval  Duration                `yaml:"refresh_interval" json:"refresh_interval"`
	DefaultView      string                  `yaml:"default_view" json:"default_view"`
	UseEmoji         bool                    `yaml:"use_emoji" json:"use_emoji"`
	UseASCIIFallback bool                    `yaml:"use_ascii_fallback" json:"use_ascii_fallback"`
	Tabs             map[string]TUITabConfig `yaml:"tabs" json:"tabs"`
}

type TUITabConfig struct {
	Enabled  bool            `yaml:"enabled" json:"enabled"`
	Title    string          `yaml:"title" json:"title"`
	Sections map[string]bool `yaml:"sections" json:"sections"`
}

func LoadEnv() (Env, error) {
	_ = godotenv.Load()
	env := Env{
		APIKey:       os.Getenv("API_KEY"),
		ConfigPath:   getenvDefault("CONFIG_PATH", "./configs/config.yaml"),
		HTTPAddr:     os.Getenv("HTTP_ADDR"),
		IBKREnabled:  os.Getenv("IBKR_ENABLED"),
		IBKRHost:     os.Getenv("IBKR_HOST"),
		IBKRPort:     os.Getenv("IBKR_PORT"),
		IBKRClientID: os.Getenv("IBKR_CLIENT_ID"),
	}
	return env, nil
}

func Load(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.setDefaults()
	return cfg, nil
}

func (c *Config) ApplyEnv(env Env) {
	if env.HTTPAddr != "" {
		c.API.HTTPAddr = env.HTTPAddr
	}
	if ibkr, ok := c.Providers["ibkr"]; ok {
		if env.IBKREnabled != "" {
			ibkr.Enabled = strings.EqualFold(env.IBKREnabled, "true") || env.IBKREnabled == "1"
		}
		if env.IBKRHost != "" {
			ibkr.Host = env.IBKRHost
		}
		if env.IBKRPort != "" {
			var port int
			_, _ = fmt.Sscanf(env.IBKRPort, "%d", &port)
			if port > 0 {
				ibkr.Port = port
			}
		}
		if env.IBKRClientID != "" {
			var clientID int
			_, _ = fmt.Sscanf(env.IBKRClientID, "%d", &clientID)
			if clientID > 0 {
				ibkr.ClientID = clientID
			}
		}
		c.Providers["ibkr"] = ibkr
	}
}

func (c *Config) KnownAssets() []string {
	seen := make(map[string]struct{})
	var out []string
	for _, list := range [][]string{c.Assets.Preferred, c.Assets.StableBases, c.Assets.QuoteAssets} {
		for _, asset := range list {
			asset = NormalizeAssetName(asset)
			if asset == "" {
				continue
			}
			if _, ok := seen[asset]; ok {
				continue
			}
			seen[asset] = struct{}{}
			out = append(out, asset)
		}
	}
	return out
}

func (c *Config) setDefaults() {
	if c.App.Version == "" {
		c.App.Version = "v2.1.0"
	}
	if c.App.RefreshInterval.Duration == 0 {
		c.App.RefreshInterval.Duration = 2 * time.Second
	}
	if c.App.StalePriceAfter.Duration == 0 {
		c.App.StalePriceAfter.Duration = 15 * time.Second
	}
	if c.MarketData.OrderBookDepth <= 0 {
		c.MarketData.OrderBookDepth = 20
	}
	if c.MarketData.OrderBookStaleAfter.Duration == 0 {
		c.MarketData.OrderBookStaleAfter.Duration = 10 * time.Second
	}
	if c.MarketData.TickerStaleAfter.Duration == 0 {
		c.MarketData.TickerStaleAfter.Duration = c.App.StalePriceAfter.Duration
	}
	if c.Simulation.QuoteAsset == "" {
		c.Simulation.QuoteAsset = "USDT"
	}
	if len(c.Simulation.TradeSizes) == 0 {
		c.Simulation.TradeSizes = []Decimal{
			MustDecimal("100"),
			MustDecimal("500"),
			MustDecimal("1000"),
			MustDecimal("5000"),
		}
	}
	if c.API.HTTPAddr == "" {
		c.API.HTTPAddr = ":8080"
	}
	c.normalizeProvidersAndExchanges()
	c.normalizeStrategies()
	if c.TUI.BackendURL == "" {
		c.TUI.BackendURL = "http://localhost:8080"
	}
	if c.TUI.RefreshInterval.Duration == 0 {
		c.TUI.RefreshInterval.Duration = c.App.RefreshInterval.Duration
	}
	if c.TUI.DefaultView == "" {
		c.TUI.DefaultView = "crypto_dashboard"
	}
	if c.TUI.Tabs == nil {
		c.TUI.Tabs = defaultTUITabs()
	}
	if c.Alerts.Cooldown.Duration == 0 {
		c.Alerts.Cooldown.Duration = 5 * time.Minute
	}
	if c.Alerts.MaxResults == 0 {
		c.Alerts.MaxResults = 50
	}
	if c.Metrics.PrometheusPath == "" {
		c.Metrics.PrometheusPath = "/metrics"
	}
	if !c.Health.ScoringEnabled {
		c.Health.ScoringEnabled = true
	}
	if c.Health.StalePenalty == 0 {
		c.Health.StalePenalty = 20
	}
	if c.Health.DisconnectedPenalty == 0 {
		c.Health.DisconnectedPenalty = 40
	}
	if c.Health.RestFallbackPenalty == 0 {
		c.Health.RestFallbackPenalty = 10
	}
	if c.Health.ReconnectPenalty == 0 {
		c.Health.ReconnectPenalty = 2
	}
	for name, ex := range c.Exchanges {
		if ex.RestPollInterval.Duration == 0 {
			ex.RestPollInterval.Duration = 5 * time.Second
			c.Exchanges[name] = ex
		}
	}
}

func (c *Config) normalizeProvidersAndExchanges() {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
	if c.Exchanges == nil {
		c.Exchanges = make(map[string]ExchangeConfig)
	}
	if len(c.Providers) == 0 {
		for name, ex := range c.Exchanges {
			c.Providers[name] = ProviderConfig{
				Enabled:          ex.Enabled,
				Type:             "crypto_exchange",
				SpotEnabled:      ex.SpotEnabled,
				FuturesEnabled:   ex.FuturesEnabled,
				WebsocketEnabled: ex.WebsocketEnabled,
				RestPollInterval: ex.RestPollInterval,
				Fees: ProviderFees{
					SpotTaker:    ex.Fees.SpotTaker,
					FuturesTaker: ex.Fees.FuturesTaker,
				},
			}
		}
	}
	for name, provider := range c.Providers {
		if provider.Type == "" {
			if strings.EqualFold(name, "ibkr") {
				provider.Type = "broker"
			} else {
				provider.Type = "crypto_exchange"
			}
		}
		if provider.RestPollInterval.Duration == 0 {
			provider.RestPollInterval.Duration = 5 * time.Second
		}
		if strings.EqualFold(provider.Type, "broker") {
			if provider.APIMode == "" {
				provider.APIMode = "tws_gateway"
			}
			if provider.Host == "" {
				provider.Host = "127.0.0.1"
			}
			if provider.Port == 0 {
				provider.Port = 7497
			}
			if provider.ClientID == 0 {
				provider.ClientID = 101
			}
			if provider.MarketDataEnabled == false && provider.Enabled {
				provider.MarketDataEnabled = true
			}
		}
		c.Providers[name] = provider
		if strings.EqualFold(provider.Type, "crypto_exchange") {
			c.Exchanges[name] = ExchangeConfig{
				Enabled:          provider.Enabled,
				SpotEnabled:      provider.SpotEnabled,
				FuturesEnabled:   provider.FuturesEnabled,
				WebsocketEnabled: provider.WebsocketEnabled,
				RestPollInterval: provider.RestPollInterval,
				Fees: ExchangeFees{
					SpotTaker:    provider.Fees.SpotTaker,
					FuturesTaker: provider.Fees.FuturesTaker,
				},
			}
		}
	}
}

func (c *Config) normalizeStrategies() {
	if c.Strategies.CryptoTriangular.Enabled || c.Strategies.CryptoTriangular.Title != "" {
		c.Arbitrage.Triangular = ResultLimitConfig{
			Enabled:           c.Strategies.CryptoTriangular.Enabled,
			MinProfitPercent:  c.Strategies.CryptoTriangular.MinProfitPercent,
			MaxResults:        c.Strategies.CryptoTriangular.MaxResults,
			UseOrderBookDepth: c.Strategies.CryptoTriangular.UseOrderBookDepth,
		}
	}
	if c.Strategies.CrossExchange.Enabled || c.Strategies.CrossExchange.Title != "" {
		c.Arbitrage.CrossExchange = ResultLimitConfig{
			Enabled:           c.Strategies.CrossExchange.Enabled,
			MinProfitPercent:  c.Strategies.CrossExchange.MinProfitPercent,
			MaxResults:        c.Strategies.CrossExchange.MaxResults,
			UseOrderBookDepth: c.Strategies.CrossExchange.UseOrderBookDepth,
		}
	}
	if c.Strategies.CryptoSpotFutures.Enabled || c.Strategies.CryptoSpotFutures.Title != "" {
		c.Arbitrage.SpotFutures = SpotFuturesArbConfig{
			Enabled:            c.Strategies.CryptoSpotFutures.Enabled,
			MinBasisPercent:    c.Strategies.CryptoSpotFutures.MinBasisPercent,
			IncludeFundingRate: c.Strategies.CryptoSpotFutures.IncludeFundingRate,
			MaxResults:         c.Strategies.CryptoSpotFutures.MaxResults,
			UseOrderBookDepth:  c.Strategies.CryptoSpotFutures.UseOrderBookDepth,
		}
	}
	if c.Strategies.CryptoTriangular.Title == "" {
		c.Strategies.CryptoTriangular.Title = "Crypto Triangular Arbitrage"
	}
	if c.Strategies.CrossExchange.Title == "" {
		c.Strategies.CrossExchange.Title = "Cross-Exchange Arbitrage"
	}
	if c.Strategies.CryptoSpotFutures.Title == "" {
		c.Strategies.CryptoSpotFutures.Title = "Crypto Spot-Futures Arbitrage"
	}
	if c.Strategies.IBKRFXTriangular.Title == "" {
		c.Strategies.IBKRFXTriangular.Title = "IBKR FX Triangular Arbitrage"
	}
	if c.Strategies.CryptoSpotVsIBKRFutures.Title == "" {
		c.Strategies.CryptoSpotVsIBKRFutures.Title = "Crypto Spot vs IBKR Futures Basis"
	}
	if c.Signals.RelatedAssets.Title == "" {
		c.Signals.RelatedAssets.Title = "Related Asset Signals"
	}
	if c.Alerts.Title == "" {
		c.Alerts.Title = "Alerts"
	}
}

func defaultTUITabs() map[string]TUITabConfig {
	return map[string]TUITabConfig{
		"crypto_dashboard": {Enabled: true, Title: "Crypto Dashboard"},
		"triangular":       {Enabled: true, Title: "Crypto Triangular"},
		"cross_exchange":   {Enabled: true, Title: "Cross-Exchange"},
		"spot_futures":     {Enabled: true, Title: "Crypto Spot-Futures"},
		"signals":          {Enabled: true, Title: "Signals"},
		"alerts":           {Enabled: true, Title: "Alerts"},
		"health":           {Enabled: true, Title: "Health"},
		"ibkr": {
			Enabled: true,
			Title:   "IBKR Monitor",
			Sections: map[string]bool{
				"instruments":                       true,
				"fx_triangular_arbitrage":           true,
				"crypto_spot_vs_ibkr_futures_basis": true,
				"health":                            true,
			},
		},
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func NormalizeAssetName(asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	switch asset {
	case "XBT":
		return "BTC"
	default:
		return asset
	}
}
