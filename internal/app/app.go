package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go-crypto-arb/internal/alerts"
	"go-crypto-arb/internal/api"
	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/broker/ibkr"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/exchange/binance"
	"go-crypto-arb/internal/exchange/kraken"
	"go-crypto-arb/internal/exchange/publicrest"
	healthpkg "go-crypto-arb/internal/health"
	"go-crypto-arb/internal/instrument"
	"go-crypto-arb/internal/marketdata"
)

type App struct {
	cfg       config.Config
	env       config.Env
	logger    *slog.Logger
	store     *marketdata.Store
	exchanges []exchange.Exchange
	brokers   []*ibkr.Client
	signals   *arbitrage.SignalEngine
	alerts    *alerts.Engine
}

func New(cfg config.Config, env config.Env, logger *slog.Logger) *App {
	if logger == nil {
		logger = slog.Default()
	}
	store := marketdata.NewStore()
	knownAssets := cfg.KnownAssets()
	var exchanges []exchange.Exchange
	for exchangeName, exchangeCfg := range cfg.Exchanges {
		if !exchangeCfg.Enabled {
			continue
		}
		switch strings.ToLower(exchangeName) {
		case "binance":
			exchanges = append(exchanges, binance.New(exchangeCfg, knownAssets, cfg.MarketData.TickerStaleAfter.Duration, cfg.MarketData.OrderBookStaleAfter.Duration, cfg.MarketData.OrderBookDepth, logger))
		case "kraken":
			exchanges = append(exchanges, kraken.New(exchangeCfg, knownAssets, cfg.MarketData.TickerStaleAfter.Duration, cfg.MarketData.OrderBookStaleAfter.Duration, cfg.MarketData.OrderBookDepth, logger))
		default:
			if client, ok := publicrest.New(exchangeName, exchangeCfg, knownAssets, cfg.MarketData.TickerStaleAfter.Duration, cfg.MarketData.OrderBookStaleAfter.Duration, cfg.MarketData.OrderBookDepth, logger); ok {
				exchanges = append(exchanges, client)
			} else {
				logger.Warn("unsupported exchange in config", "exchange", exchangeName)
			}
		}
	}
	var brokers []*ibkr.Client
	if providerCfg, ok := cfg.Providers["ibkr"]; ok && providerCfg.Enabled {
		brokers = append(brokers, ibkr.New(providerCfg, instrument.IBKRInstruments(cfg), logger))
	}
	return &App{
		cfg:       cfg,
		env:       env,
		logger:    logger,
		store:     store,
		exchanges: exchanges,
		brokers:   brokers,
		signals:   arbitrage.NewSignalEngine(30),
		alerts:    alerts.NewEngine(),
	}
}

func (a *App) Run(ctx context.Context) error {
	for _, ex := range a.exchanges {
		if err := ex.Start(ctx); err != nil {
			return err
		}
	}
	for _, broker := range a.brokers {
		if err := broker.Start(ctx); err != nil {
			return err
		}
	}
	a.calculate()
	go a.calculationLoop(ctx)

	httpServer := &http.Server{
		Addr:              a.cfg.API.HTTPAddr,
		Handler:           api.NewServer(a.cfg, a.store, a.env.APIKey, a.logger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("api listening", "addr", a.cfg.API.HTTPAddr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, ex := range a.exchanges {
		_ = ex.Stop(shutdownCtx)
	}
	for _, broker := range a.brokers {
		_ = broker.Stop(shutdownCtx)
	}
	return httpServer.Shutdown(shutdownCtx)
}

func (a *App) calculationLoop(ctx context.Context) {
	interval := a.cfg.App.RefreshInterval.Duration
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.calculate()
		}
	}
}

func (a *App) calculate() {
	var spot []exchange.Ticker
	var futures []exchange.Ticker
	var funding []exchange.FundingRate
	var orderBooks []exchange.OrderBook
	var markets []exchange.MarketInfo
	var health []exchange.ExchangeHealth
	var brokerTickers []exchange.Ticker

	for _, ex := range a.exchanges {
		spot = append(spot, ex.GetLatestTickers()...)
		futures = append(futures, ex.GetLatestFuturesTickers()...)
		funding = append(funding, ex.GetFundingRates()...)
		orderBooks = append(orderBooks, ex.GetLatestOrderBooks()...)
		markets = append(markets, ex.GetMarkets()...)
		health = append(health, healthpkg.Score(a.cfg.Health, ex.Health()))
	}
	for _, broker := range a.brokers {
		brokerTickers = append(brokerTickers, broker.GetLatestTickers()...)
		orderBooks = append(orderBooks, broker.GetLatestOrderBooks()...)
		markets = append(markets, broker.GetMarkets()...)
		health = append(health, healthpkg.Score(a.cfg.Health, broker.Health()))
	}
	a.store.UpsertSpotTickers(spot)
	a.store.UpsertFuturesTickers(futures)
	a.store.UpsertFundingRates(funding)
	a.store.UpsertOrderBooks(orderBooks)
	a.store.SetMarkets(markets)
	a.store.SetExchangeHealth(health)

	triangular := arbitrage.CalculateTriangularV2(a.cfg, spot, orderBooks)
	cross := arbitrage.CalculateCrossExchangeV2(a.cfg, spot, orderBooks)
	spotFutures := arbitrage.CalculateSpotFuturesV2(a.cfg, spot, futures, orderBooks, funding)
	ibkrFX := arbitrage.CalculateIBKRFXTriangular(a.cfg, brokerTickers, orderBooks)
	brokerBasis := arbitrage.CalculateBrokerFuturesBasis(a.cfg, spot, brokerTickers, orderBooks)
	related := a.signals.Update(a.cfg, spot)
	a.store.SetCalculations(triangular, cross, spotFutures, related)
	a.store.SetBrokerCalculations(ibkrFX, brokerBasis)
	a.store.SetAlerts(a.alerts.Evaluate(a.cfg, triangular, cross, spotFutures, ibkrFX, brokerBasis, health))
}
