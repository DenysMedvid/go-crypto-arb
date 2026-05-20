package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/exchange"
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	plainStyle  = lipgloss.NewStyle()
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

const (
	minTerminalWidth = 40
	cardGapWidth     = 1

	headerCardWidth      = 104
	tabsCardWidth        = 104
	priceCardWidth       = 58
	opportunityCardWidth = 100
	signalsCardWidth     = 66
	alertsCardWidth      = 66
	healthCardWidth      = 92
	ibkrHeaderCardWidth  = 96
	ibkrTableCardWidth   = 100
	ibkrHealthCardWidth  = 88
	helpCardWidth        = 52
	footerCardWidth      = 104
	detailCardWidth      = 104
)

func (m Model) View() string {
	width := m.width
	if width < minTerminalWidth {
		width = minTerminalWidth
	}
	var sections []string
	sections = append(sections, m.renderHeader(width))
	sections = append(sections, m.renderTabs(width))
	switch m.currentView {
	case "triangular":
		sections = append(sections, m.renderTriangular(width))
	case "cross_exchange":
		sections = append(sections, m.renderCrossExchange(width))
	case "spot_futures":
		sections = append(sections, m.renderSpotFutures(width))
	case "signals":
		sections = append(sections, m.renderSignals(width))
	case "alerts":
		sections = append(sections, m.renderAlerts(width))
	case "health":
		sections = append(sections, m.renderHealth(width))
	case "ibkr":
		sections = append(sections, m.renderIBKR(width))
	default:
		sections = append(sections, m.renderPrices(width))
		sections = append(sections, flowCards(width, m.renderTriangular(width), m.renderCrossExchange(width), m.renderSpotFutures(width)))
		sections = append(sections, m.renderSignalsAndAlerts(width))
		sections = append(sections, m.renderHealth(width))
	}
	if m.detailOpen {
		sections = append(sections, m.renderDetail(width))
	}
	if m.showHelp {
		sections = append(sections, m.renderHelp(width))
	}
	sections = append(sections, m.renderFooter(width))
	return strings.Join(sections, "\n")
}

func (m Model) renderHeader(width int) string {
	status := greenStyle.Render("OK")
	if m.err != nil {
		status = redStyle.Render("ERR")
	}
	last := "never"
	if !m.lastRefresh.IsZero() {
		last = m.lastRefresh.Format("15:04:05")
	}
	paused := ""
	if m.paused {
		paused = " | paused"
	}
	line := fmt.Sprintf("Backend: %s | API: %s | Last refresh: %s | Latency: %s | q quit%s", status, m.client.BaseURL(), last, shortDuration(m.latency), paused)
	cardWidth := fixedCardWidth(width, headerCardWidth)
	return card(m.icons.App+" go-crypto-arb", truncate(line, max(20, cardWidth-6)), cardWidth)
}

func (m Model) renderTabs(width int) string {
	cardWidth := fixedCardWidth(width, tabsCardWidth)
	tabs := []struct {
		key   string
		view  string
		title string
	}{
		{"1", "crypto_dashboard", titleOrDefault(m.tabTitles, "crypto_dashboard", "Crypto Dashboard")},
		{"2", "triangular", titleOrDefault(m.tabTitles, "triangular", "Crypto Triangular")},
		{"3", "cross_exchange", titleOrDefault(m.tabTitles, "cross_exchange", "Cross-Exchange")},
		{"4", "spot_futures", titleOrDefault(m.tabTitles, "spot_futures", "Crypto Spot-Futures")},
		{"5", "signals", titleOrDefault(m.tabTitles, "signals", "Signals")},
		{"6", "alerts", titleOrDefault(m.tabTitles, "alerts", "Alerts")},
		{"7", "health", titleOrDefault(m.tabTitles, "health", "Health")},
		{"8", "ibkr", titleOrDefault(m.tabTitles, "ibkr", "IBKR Monitor")},
	}
	var parts []string
	for _, tab := range tabs {
		label := fmt.Sprintf("%s %s", tab.key, tab.title)
		if m.currentView == tab.view {
			parts = append(parts, greenStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, mutedStyle.Render(label))
		}
	}
	return card("Tabs", truncate(strings.Join(parts, "  "), max(20, cardWidth-6)), cardWidth)
}

func (m Model) renderPrices(width int) string {
	names := sortedPriceExchanges(m.snapshot.Prices)
	if len(names) == 0 {
		return card(m.icons.Prices+" Prices", mutedStyle.Render("No spot prices yet"), fixedCardWidth(width, priceCardWidth))
	}
	var cards []string
	highlights := m.priceHighlightStats(m.snapshot.Prices)
	cardWidth := fixedCardWidth(width, priceCardWidth)
	for _, name := range names {
		cards = append(cards, m.renderPriceCard(name, m.snapshot.Prices[name], cardWidth, highlights))
	}
	return flowCards(width, cards...)
}

func (m Model) renderPriceCard(exchangeName string, tickers []exchange.Ticker, width int, highlights priceHighlightStats) string {
	tickers = m.filterDashboardPrices(tickers)
	lines := []string{fmt.Sprintf("%-12s %12s %12s %6s", "Symbol", "Bid", "Ask", "Age")}
	limit := min(len(tickers), 8)
	for i := 0; i < limit; i++ {
		t := tickers[i]
		age := ageString(t.UpdatedAt)
		baseStyle := plainStyle
		if time.Since(t.UpdatedAt) > 15*time.Second {
			baseStyle = yellowStyle
		}
		symbol := exchange.NormalizeCanonicalSymbol(t.Symbol)
		bidStyle := priceHighlightStyle(highlights.bidHighlight(symbol, t.Bid), baseStyle)
		askStyle := priceHighlightStyle(highlights.askHighlight(symbol, t.Ask), baseStyle)
		lines = append(lines, renderPriceRow(t.Symbol, money(t.Bid), money(t.Ask), age, baseStyle, bidStyle, askStyle, baseStyle))
	}
	if len(tickers) > limit {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("... %d more", len(tickers)-limit)))
	}
	if len(tickers) == 0 {
		lines = append(lines, mutedStyle.Render("No configured spot prices yet"))
	}
	return card(m.icons.Prices+" Prices: "+exchangeName, strings.Join(lines, "\n"), width)
}

func (m Model) filterDashboardPrices(tickers []exchange.Ticker) []exchange.Ticker {
	if len(m.priceSymbols) == 0 {
		out := append([]exchange.Ticker(nil), tickers...)
		sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
		return out
	}
	out := make([]exchange.Ticker, 0, len(tickers))
	for _, ticker := range tickers {
		symbol := exchange.NormalizeCanonicalSymbol(ticker.Symbol)
		if _, ok := m.priceSymbols[symbol]; ok {
			out = append(out, ticker)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left := m.priceSymbols[exchange.NormalizeCanonicalSymbol(out[i].Symbol)]
		right := m.priceSymbols[exchange.NormalizeCanonicalSymbol(out[j].Symbol)]
		if left == right {
			return out[i].Symbol < out[j].Symbol
		}
		return left < right
	})
	return out
}

func (m Model) renderTriangular(width int) string {
	cardWidth := fixedCardWidth(width, opportunityCardWidth)
	items := m.filterTriangular(m.snapshot.TriangularArbitrage)
	lines := []string{renderTriangularHeader()}
	for i, item := range first(items, 8) {
		lines = append(lines, renderTriangularRow(m.selectMarker(i), item))
	}
	if len(items) == 0 {
		lines = append(lines, mutedStyle.Render("No triangular results yet"))
	}
	return card(m.icons.Triangular+" "+titleOrDefault(m.tabTitles, "triangular", "Triangular Arbitrage"), strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderCrossExchange(width int) string {
	cardWidth := fixedCardWidth(width, opportunityCardWidth)
	items := m.filterCross(m.snapshot.CrossExchangeArbitrage)
	offset := 0
	if m.currentView == "crypto_dashboard" || m.currentView == "" {
		offset = len(m.filterTriangular(m.snapshot.TriangularArbitrage))
	}
	lines := []string{renderCrossExchangeHeader()}
	for i, item := range first(items, 8) {
		idx := offset + i
		lines = append(lines, renderCrossExchangeRow(m.selectMarker(idx), item))
	}
	if len(items) == 0 {
		lines = append(lines, mutedStyle.Render("No cross-exchange results yet"))
	}
	return card(m.icons.CrossExchange+" "+titleOrDefault(m.tabTitles, "cross_exchange", "Cross-Exchange Arbitrage"), strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderSpotFutures(width int) string {
	cardWidth := fixedCardWidth(width, opportunityCardWidth)
	items := m.filterSpotFutures(m.snapshot.SpotFuturesArbitrage)
	offset := 0
	if m.currentView == "crypto_dashboard" || m.currentView == "" {
		offset = len(m.filterTriangular(m.snapshot.TriangularArbitrage)) + len(m.filterCross(m.snapshot.CrossExchangeArbitrage))
	}
	lines := []string{renderSpotFuturesHeader()}
	for i, item := range first(items, 8) {
		idx := offset + i
		lines = append(lines, renderSpotFuturesRow(m.selectMarker(idx), item))
	}
	if len(items) == 0 {
		lines = append(lines, mutedStyle.Render("No spot-futures results yet"))
	}
	return card(m.icons.SpotFutures+" "+titleOrDefault(m.tabTitles, "spot_futures", "Spot-Futures Arbitrage"), strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderSignalsAndAlerts(width int) string {
	return flowCards(width, m.renderSignals(width), m.renderAlerts(width))
}

func (m Model) renderSignals(width int) string {
	cardWidth := fixedCardWidth(width, signalsCardWidth)
	var lines []string
	for _, group := range first(m.snapshot.RelatedAssetSignals, 4) {
		lines = append(lines, titleStyle.Render(truncate(group.Group, max(10, cardWidth-8))))
		for _, item := range first(group.Assets, 5) {
			lines = append(lines, fmt.Sprintf("%-10s %9s divergence %9s", truncate(item.Symbol, 10), coloredPercent(item.ChangePercent), coloredPercent(item.DivergencePercent)))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, mutedStyle.Render("No related asset signals yet"))
	}
	return card(m.icons.Signals+" "+titleOrDefault(m.tabTitles, "signals", "Related Asset Signals"), strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderAlerts(width int) string {
	cardWidth := fixedCardWidth(width, alertsCardWidth)
	var lines []string
	for _, alert := range first(m.snapshot.Alerts, 8) {
		stamp := alert.CreatedAt.Format("15:04:05")
		line := truncate(fmt.Sprintf("[%s] %s", stamp, alert.Message), max(10, cardWidth-6))
		if string(alert.Severity) == "critical" {
			line = redStyle.Render(line)
		} else if string(alert.Severity) == "warning" {
			line = yellowStyle.Render(line)
		}
		if alert.RepeatCount > 0 {
			line += mutedStyle.Render(fmt.Sprintf(" x%d", alert.RepeatCount+1))
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, mutedStyle.Render("No alerts"))
	}
	return card(m.icons.Alerts+" "+titleOrDefault(m.tabTitles, "alerts", "Alerts"), strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderHealth(width int) string {
	cardWidth := fixedCardWidth(width, healthCardWidth)
	names := make([]string, 0, len(m.snapshot.ExchangeHealth))
	for name := range m.snapshot.ExchangeHealth {
		names = append(names, name)
	}
	sort.Strings(names)
	lines := []string{renderHealthHeader()}
	for _, name := range names {
		h := m.snapshot.ExchangeHealth[name]
		lines = append(lines, renderHealthRow(h))
		if h.LastError != "" {
			lines = append(lines, yellowStyle.Render("  "+truncate(h.LastError, max(20, cardWidth-8))))
		}
	}
	if len(names) == 0 {
		lines = append(lines, mutedStyle.Render("No exchange health yet"))
	}
	return card(m.icons.Health+" "+titleOrDefault(m.tabTitles, "health", "Health"), strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderIBKR(width int) string {
	return flowCards(
		width,
		m.renderIBKRHeader(width),
		m.renderIBKRInstruments(width),
		m.renderIBKRFX(width),
		m.renderIBKRBasis(width),
		m.renderIBKRHealth(width),
	)
}

func (m Model) renderIBKRHeader(width int) string {
	cardWidth := fixedCardWidth(width, ibkrHeaderCardWidth)
	h := m.ibkrHealth()
	status := m.icons.Warning + " WARN"
	if h.Status == "ok" {
		status = m.icons.OK + " OK"
	} else if h.Status == "disconnected" || h.LastError != "" {
		status = m.icons.Error + " ERROR"
	}
	trading := m.icons.Locked + " Trading: DISABLED"
	if h.TradingEnabled {
		trading = m.icons.Error + " Trading: UNSUPPORTED"
	}
	body := fmt.Sprintf("Status: %s | %s Market Data Only | %s | Age: %s", status, m.icons.MarketDataOnly, trading, ageString(h.LastMessageTime))
	return card(m.icons.IBKR+" "+titleOrDefault(m.tabTitles, "ibkr", "IBKR Monitor"), truncate(body, max(20, cardWidth-6)), cardWidth)
}

func (m Model) renderIBKRInstruments(width int) string {
	cardWidth := fixedCardWidth(width, ibkrTableCardWidth)
	items := append([]exchange.MarketInfo(nil), m.snapshot.IBKRInstruments...)
	sort.Slice(items, func(i, j int) bool { return items[i].DisplayName < items[j].DisplayName })
	books := m.ibkrBookLookup()
	lines := []string{fmt.Sprintf("  %-28s %-8s %-10s %12s %12s %12s %6s", "Title", "Type", "Symbol", "Bid", "Ask", "Last", "Age")}
	for i, item := range first(items, 10) {
		book := books[item.InstrumentID]
		bid, ask, last, age := "n/a", "n/a", "n/a", "n/a"
		if len(book.Bids) > 0 {
			bid = money(book.Bids[0].Price)
		}
		if len(book.Asks) > 0 {
			ask = money(book.Asks[0].Price)
		}
		if len(book.Bids) > 0 && len(book.Asks) > 0 {
			last = money(book.Bids[0].Price.Add(book.Asks[0].Price).Div(decimal.NewFromInt(2)))
		}
		if !book.UpdatedAt.IsZero() {
			age = ageString(book.UpdatedAt)
		}
		lines = append(lines, fmt.Sprintf("%s %-28s %-8s %-10s %12s %12s %12s %6s",
			m.selectMarker(i), truncate(displayMarketName(item), 28), truncate(strings.ToUpper(item.AssetClass), 8), truncate(item.Symbol, 10), bid, ask, last, age))
	}
	if len(items) == 0 {
		lines = append(lines, mutedStyle.Render("No IBKR instruments configured"))
	}
	return card("IBKR Instruments", strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderIBKRFX(width int) string {
	cardWidth := fixedCardWidth(width, ibkrTableCardWidth)
	offset := len(m.snapshot.IBKRInstruments)
	items := m.filterTriangular(m.snapshot.IBKRFXTriangular)
	lines := []string{fmt.Sprintf("  %-26s %10s %12s %9s %-10s", "Cycle", "Size", "End Amount", "Net %", "Status")}
	for i, item := range first(items, 8) {
		idx := offset + i
		status := m.watchStatus(item.NetProfitPercent, item.CompleteFill)
		lines = append(lines, fmt.Sprintf("%s %-26s %10s %12s %9s %-10s",
			m.selectMarker(idx), truncate(strings.Join(item.Cycle, " -> "), 26), item.StartAmount.StringFixed(0), item.EndAmount.StringFixed(4), coloredPercent(item.NetProfitPercent), status))
	}
	if len(items) == 0 {
		lines = append(lines, mutedStyle.Render("No IBKR FX triangular results yet"))
	}
	return card("IBKR FX Triangular Arbitrage", strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderIBKRBasis(width int) string {
	cardWidth := fixedCardWidth(width, ibkrTableCardWidth)
	offset := len(m.snapshot.IBKRInstruments) + len(m.filterTriangular(m.snapshot.IBKRFXTriangular))
	items := m.filterBrokerBasis(m.snapshot.CryptoSpotVsIBKRBasis)
	lines := []string{fmt.Sprintf("  %-6s %-12s %12s %-14s %12s %9s %-10s", "Asset", "Spot Source", "Spot Ask", "IBKR Future", "Fut Bid", "Basis %", "Status")}
	for i, item := range first(items, 8) {
		idx := offset + i
		status := m.watchStatus(item.NetEstimatePercent, item.CompleteFill)
		lines = append(lines, fmt.Sprintf("%s %-6s %-12s %12s %-14s %12s %9s %-10s",
			m.selectMarker(idx), truncate(item.Asset, 6), truncate(item.SpotProvider, 12), money(item.SpotAsk), truncate(item.FuturesInstrumentID, 14), money(item.FuturesBid), coloredPercent(item.BasisPercent), status))
	}
	if len(items) == 0 {
		lines = append(lines, mutedStyle.Render("No crypto spot vs IBKR futures basis results yet"))
	}
	return card("Crypto Spot vs IBKR Futures Basis", strings.Join(lines, "\n"), cardWidth)
}

func (m Model) renderIBKRHealth(width int) string {
	cardWidth := fixedCardWidth(width, ibkrHealthCardWidth)
	h := m.ibkrHealth()
	gateway := m.icons.Error + " Disconnected"
	if h.GatewayConnected {
		gateway = m.icons.OK + " Connected"
	}
	data := m.icons.Warning + " WARN"
	if h.MarketDataOK {
		data = m.icons.OK + " OK"
	}
	errText := h.LastError
	if errText == "" {
		errText = "none"
	}
	body := fmt.Sprintf("Gateway: %s | Market Data: %s | Reconnects: %d | Score: %s | Error: %s", gateway, data, h.ReconnectCount, healthScore(h.Score), truncate(errText, max(20, cardWidth-50)))
	return card("IBKR Health", truncate(body, max(20, cardWidth-6)), cardWidth)
}

func (m Model) renderHelp(width int) string {
	body := strings.Join([]string{
		"q / Ctrl+C  quit",
		"r           refresh now",
		"p           pause or resume auto-refresh",
		"f           toggle profitable-only rows",
		"up/down     select opportunity",
		"enter       open detail panel",
		"esc         close detail panel",
		"?           toggle help",
		"1-8         switch tabs",
	}, "\n")
	return card("Help", body, fixedCardWidth(width, helpCardWidth))
}

func (m Model) renderFooter(width int) string {
	cardWidth := fixedCardWidth(width, footerCardWidth)
	mode := "all"
	if m.profitableOnly {
		mode = "profitable-only"
	}
	body := fmt.Sprintf("Controls: q/Ctrl+C quit | r refresh | p pause | f profitable-only | up/down select | Enter detail | Esc close | ? help | 1-8 tabs | mode: %s", mode)
	if m.err != nil {
		body += " | " + redStyle.Render(truncate(m.err.Error(), max(20, cardWidth-10)))
	}
	return card("Footer", truncate(body, max(20, cardWidth-6)), cardWidth)
}

func (m Model) renderDetail(width int) string {
	cardWidth := fixedCardWidth(width, detailCardWidth)
	rows := m.opportunityRows()
	if len(rows) == 0 {
		return card("Detail", mutedStyle.Render("No opportunity selected"), cardWidth)
	}
	row := rows[clampSelection(m.selectedRow, len(rows))]
	switch item := row.value.(type) {
	case arbitrage.TriangularOpportunityV2:
		lines := []string{
			fmt.Sprintf("Exchange: %s", item.Exchange),
			fmt.Sprintf("Cycle: %s", strings.Join(item.Cycle, " -> ")),
			fmt.Sprintf("Trade size: %s %s", item.StartAmount.String(), item.StartAsset),
			fmt.Sprintf("Start: %s | End: %s | Net: %s", item.StartAmount.StringFixed(4), item.EndAmount.StringFixed(4), coloredPercent(item.NetProfitPercent)),
			fmt.Sprintf("Complete fill: %s | Worst leg: %s | Max slippage: %s", fillStatus(item.CompleteFill), item.WorstLeg, coloredPercent(item.MaxSlippagePercent)),
			fmt.Sprintf("%-6s %-6s %-10s %-5s %12s %12s %12s %10s %9s %-8s", "From", "To", "Symbol", "Side", "Input", "Output", "Avg Price", "Fee", "Slip", "Fill"),
		}
		for _, leg := range item.Legs {
			lines = append(lines, fmt.Sprintf("%-6s %-6s %-10s %-5s %12s %12s %12s %10s %9s %-8s",
				leg.FromAsset, leg.ToAsset, truncate(leg.Symbol, 10), leg.Side, leg.InputAmount.StringFixed(4), leg.OutputAmount.StringFixed(4), money(leg.AveragePrice), leg.FeeAmount.StringFixed(6), coloredPercent(leg.SlippagePercent), fillStatus(leg.CompleteFill)))
		}
		return card("Opportunity Detail: Triangular", strings.Join(lines, "\n"), cardWidth)
	case arbitrage.CrossExchangeOpportunityV2:
		lines := []string{
			fmt.Sprintf("Symbol: %s | Trade size: %s", item.Symbol, item.TradeSize.String()),
			fmt.Sprintf("Buy: %s avg %s slippage %s fee %s", item.BuyExchange, money(item.BuyAveragePrice), coloredPercent(item.BuySlippagePercent), item.BuyFeeAmount.StringFixed(6)),
			fmt.Sprintf("Sell: %s avg %s slippage %s fee %s", item.SellExchange, money(item.SellAveragePrice), coloredPercent(item.SellSlippagePercent), item.SellFeeAmount.StringFixed(6)),
			fmt.Sprintf("Net profit: %s | Complete fill: %s", coloredPercent(item.NetProfitPercent), fillStatus(item.CompleteFill)),
		}
		return card("Opportunity Detail: Cross-Exchange", strings.Join(lines, "\n"), cardWidth)
	case arbitrage.SpotFuturesOpportunityV2:
		lines := []string{
			fmt.Sprintf("Exchange: %s | Symbol: %s | Trade size: %s", item.Exchange, item.Symbol, item.TradeSize.String()),
			fmt.Sprintf("Spot avg buy: %s | Futures avg sell: %s", money(item.SpotAverageBuyPrice), money(item.FuturesAverageSellPrice)),
			fmt.Sprintf("Basis: %s | Funding: %s | Net estimate: %s", coloredPercent(item.BasisPercent), coloredPercent(item.FundingRate.Mul(decimal.NewFromInt(100))), coloredPercent(item.NetEstimatePercent)),
			fmt.Sprintf("Spot slippage: %s | Futures slippage: %s | Complete fill: %s", coloredPercent(item.SpotSlippagePercent), coloredPercent(item.FuturesSlippagePercent), fillStatus(item.CompleteFill)),
		}
		return card("Opportunity Detail: Spot-Futures", strings.Join(lines, "\n"), cardWidth)
	case arbitrage.BrokerFuturesBasisOpportunity:
		lines := []string{
			fmt.Sprintf("Asset: %s | Strategy: %s", item.Asset, item.StrategyTitle),
			fmt.Sprintf("Spot: %s %s ask %s", item.SpotProvider, item.SpotSymbol, money(item.SpotAsk)),
			fmt.Sprintf("Future: %s %s bid %s", item.FuturesProvider, item.FuturesInstrumentID, money(item.FuturesBid)),
			fmt.Sprintf("Basis: %s | Net estimate: %s | Complete fill: %s", coloredPercent(item.BasisPercent), coloredPercent(item.NetEstimatePercent), fillStatus(item.CompleteFill)),
			"Trading status: DISABLED",
		}
		return card("Opportunity Detail: IBKR Futures Basis", strings.Join(lines, "\n"), cardWidth)
	case exchange.MarketInfo:
		book := m.ibkrBookLookup()[item.InstrumentID]
		bid, ask, last, age := "n/a", "n/a", "n/a", "n/a"
		if len(book.Bids) > 0 {
			bid = money(book.Bids[0].Price)
		}
		if len(book.Asks) > 0 {
			ask = money(book.Asks[0].Price)
		}
		if len(book.Bids) > 0 && len(book.Asks) > 0 {
			last = money(book.Bids[0].Price.Add(book.Asks[0].Price).Div(decimal.NewFromInt(2)))
		}
		if !book.UpdatedAt.IsZero() {
			age = ageString(book.UpdatedAt)
		}
		lines := []string{
			fmt.Sprintf("Title: %s", displayMarketName(item)),
			fmt.Sprintf("Instrument ID: %s | Symbol: %s", item.InstrumentID, item.Symbol),
			fmt.Sprintf("Type: %s | Exchange: %s | Currency: %s", strings.ToUpper(item.AssetClass), item.Exchange, item.QuoteAsset),
			fmt.Sprintf("Bid: %s | Ask: %s | Last: %s | Age: %s", bid, ask, last, age),
			fmt.Sprintf("Provider health: %s | Trading status: DISABLED", m.ibkrHealth().Status),
		}
		return card("IBKR Detail", strings.Join(lines, "\n"), cardWidth)
	default:
		return card("Detail", mutedStyle.Render("Unknown opportunity type"), cardWidth)
	}
}

func card(title, body string, width int) string {
	contentWidth := max(20, width-2)
	return borderStyle.Width(contentWidth).Render(titleStyle.Render(title) + "\n" + body)
}

func fixedCardWidth(viewportWidth int, preferredWidth int) int {
	if viewportWidth < minTerminalWidth {
		viewportWidth = minTerminalWidth
	}
	if preferredWidth <= 0 || preferredWidth > viewportWidth {
		return viewportWidth
	}
	return preferredWidth
}

func flowCards(width int, cards ...string) string {
	if len(cards) == 0 {
		return ""
	}
	gap := strings.Repeat(" ", cardGapWidth)
	var rows []string
	var row []string
	rowWidth := 0
	for _, renderedCard := range cards {
		cardWidth := lipgloss.Width(renderedCard)
		nextWidth := cardWidth
		if len(row) > 0 {
			nextWidth = rowWidth + cardGapWidth + cardWidth
		}
		if len(row) > 0 && nextWidth > width {
			rows = append(rows, joinCardRow(gap, row))
			row = nil
			rowWidth = 0
		}
		if len(row) > 0 {
			rowWidth += cardGapWidth + cardWidth
		} else {
			rowWidth = cardWidth
		}
		row = append(row, renderedCard)
	}
	if len(row) > 0 {
		rows = append(rows, joinCardRow(gap, row))
	}
	return strings.Join(rows, "\n")
}

func joinCardRow(gap string, cards []string) string {
	if len(cards) == 1 {
		return cards[0]
	}
	parts := make([]string, 0, len(cards)*2-1)
	for i, renderedCard := range cards {
		if i > 0 {
			parts = append(parts, gap)
		}
		parts = append(parts, renderedCard)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

type opportunityRow struct {
	kind  string
	value any
}

func (m Model) opportunityRows() []opportunityRow {
	var rows []opportunityRow
	if m.currentView == "ibkr" {
		for _, item := range m.snapshot.IBKRInstruments {
			rows = append(rows, opportunityRow{kind: "ibkr_instrument", value: item})
		}
		for _, item := range m.filterTriangular(m.snapshot.IBKRFXTriangular) {
			rows = append(rows, opportunityRow{kind: "ibkr_fx", value: item})
		}
		for _, item := range m.filterBrokerBasis(m.snapshot.CryptoSpotVsIBKRBasis) {
			rows = append(rows, opportunityRow{kind: "ibkr_basis", value: item})
		}
		return rows
	}
	if m.currentView == "triangular" || m.currentView == "crypto_dashboard" || m.currentView == "" {
		for _, item := range m.filterTriangular(m.snapshot.TriangularArbitrage) {
			rows = append(rows, opportunityRow{kind: "triangular", value: item})
		}
	}
	if m.currentView == "cross_exchange" || m.currentView == "crypto_dashboard" || m.currentView == "" {
		for _, item := range m.filterCross(m.snapshot.CrossExchangeArbitrage) {
			rows = append(rows, opportunityRow{kind: "cross_exchange", value: item})
		}
	}
	if m.currentView == "spot_futures" || m.currentView == "crypto_dashboard" || m.currentView == "" {
		for _, item := range m.filterSpotFutures(m.snapshot.SpotFuturesArbitrage) {
			rows = append(rows, opportunityRow{kind: "spot_futures", value: item})
		}
	}
	return rows
}

func (m Model) allCryptoOpportunityRows() []opportunityRow {
	var rows []opportunityRow
	for _, item := range m.filterTriangular(m.snapshot.TriangularArbitrage) {
		rows = append(rows, opportunityRow{kind: "triangular", value: item})
	}
	for _, item := range m.filterCross(m.snapshot.CrossExchangeArbitrage) {
		rows = append(rows, opportunityRow{kind: "cross_exchange", value: item})
	}
	for _, item := range m.filterSpotFutures(m.snapshot.SpotFuturesArbitrage) {
		rows = append(rows, opportunityRow{kind: "spot_futures", value: item})
	}
	return rows
}

func (m Model) opportunityCount() int {
	return len(m.opportunityRows())
}

func (m Model) selectMarker(index int) string {
	if m.selectedRow == index {
		return greenStyle.Render(">")
	}
	return " "
}

func (m Model) filterTriangular(items []arbitrage.TriangularOpportunityV2) []arbitrage.TriangularOpportunityV2 {
	if !m.profitableOnly {
		return items
	}
	var out []arbitrage.TriangularOpportunityV2
	for _, item := range items {
		if item.NetProfitPercent.GreaterThan(decimal.Zero) {
			out = append(out, item)
		}
	}
	return out
}

type priceHighlight int

const (
	priceHighlightNone priceHighlight = iota
	priceHighlightBest
	priceHighlightWorst
)

type priceHighlightStats struct {
	bids map[string]priceExtrema
	asks map[string]priceExtrema
}

type priceExtrema struct {
	best      decimal.Decimal
	worst     decimal.Decimal
	hasValue  bool
	hasSpread bool
}

func (m Model) priceHighlightStats(prices map[string][]exchange.Ticker) priceHighlightStats {
	stats := priceHighlightStats{
		bids: make(map[string]priceExtrema),
		asks: make(map[string]priceExtrema),
	}
	for _, tickers := range prices {
		for _, ticker := range m.filterDashboardPrices(tickers) {
			symbol := exchange.NormalizeCanonicalSymbol(ticker.Symbol)
			updatePriceExtrema(stats.bids, symbol, ticker.Bid,
				func(left, right decimal.Decimal) bool { return left.GreaterThan(right) },
				func(left, right decimal.Decimal) bool { return left.LessThan(right) })
			updatePriceExtrema(stats.asks, symbol, ticker.Ask,
				func(left, right decimal.Decimal) bool { return left.LessThan(right) },
				func(left, right decimal.Decimal) bool { return left.GreaterThan(right) })
		}
	}
	finalizePriceExtrema(stats.bids)
	finalizePriceExtrema(stats.asks)
	return stats
}

func updatePriceExtrema(extrema map[string]priceExtrema, symbol string, value decimal.Decimal, better func(decimal.Decimal, decimal.Decimal) bool, worse func(decimal.Decimal, decimal.Decimal) bool) {
	if symbol == "" || !value.GreaterThan(decimal.Zero) {
		return
	}
	current := extrema[symbol]
	if !current.hasValue {
		extrema[symbol] = priceExtrema{best: value, worst: value, hasValue: true}
		return
	}
	if better(value, current.best) {
		current.best = value
	}
	if worse(value, current.worst) {
		current.worst = value
	}
	extrema[symbol] = current
}

func finalizePriceExtrema(extrema map[string]priceExtrema) {
	for symbol, current := range extrema {
		current.hasSpread = current.hasValue && !current.best.Equal(current.worst)
		extrema[symbol] = current
	}
}

func (stats priceHighlightStats) bidHighlight(symbol string, value decimal.Decimal) priceHighlight {
	return priceHighlightFor(stats.bids, symbol, value)
}

func (stats priceHighlightStats) askHighlight(symbol string, value decimal.Decimal) priceHighlight {
	return priceHighlightFor(stats.asks, symbol, value)
}

func priceHighlightFor(extrema map[string]priceExtrema, symbol string, value decimal.Decimal) priceHighlight {
	if !value.GreaterThan(decimal.Zero) {
		return priceHighlightNone
	}
	current, ok := extrema[symbol]
	if !ok || !current.hasSpread {
		return priceHighlightNone
	}
	if value.Equal(current.best) {
		return priceHighlightBest
	}
	if value.Equal(current.worst) {
		return priceHighlightWorst
	}
	return priceHighlightNone
}

func (m Model) filterCross(items []arbitrage.CrossExchangeOpportunityV2) []arbitrage.CrossExchangeOpportunityV2 {
	out := make([]arbitrage.CrossExchangeOpportunityV2, 0, len(items))
	for _, item := range items {
		if !m.profitableOnly || item.NetProfitPercent.GreaterThan(decimal.Zero) {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].NetProfitPercent.Equal(out[j].NetProfitPercent) {
			return out[i].NetProfitPercent.GreaterThan(out[j].NetProfitPercent)
		}
		leftSpread := out[i].SellAveragePrice.Sub(out[i].BuyAveragePrice)
		rightSpread := out[j].SellAveragePrice.Sub(out[j].BuyAveragePrice)
		if !leftSpread.Equal(rightSpread) {
			return leftSpread.GreaterThan(rightSpread)
		}
		return out[i].Symbol < out[j].Symbol
	})
	return out
}

func (m Model) filterSpotFutures(items []arbitrage.SpotFuturesOpportunityV2) []arbitrage.SpotFuturesOpportunityV2 {
	if !m.profitableOnly {
		return items
	}
	var out []arbitrage.SpotFuturesOpportunityV2
	for _, item := range items {
		if item.NetEstimatePercent.GreaterThan(decimal.Zero) {
			out = append(out, item)
		}
	}
	return out
}

func sortedPriceExchanges(prices map[string][]exchange.Ticker) []string {
	names := make([]string, 0, len(prices))
	for name := range prices {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func money(value decimal.Decimal) string {
	if value.Abs().GreaterThan(decimal.NewFromInt(1000)) {
		return value.StringFixed(2)
	}
	return value.StringFixed(4)
}

func renderPriceRow(symbol string, bid string, ask string, age string, symbolStyle lipgloss.Style, bidStyle lipgloss.Style, askStyle lipgloss.Style, ageStyle lipgloss.Style) string {
	return strings.Join([]string{
		renderCell(truncate(symbol, 12), 12, false, symbolStyle),
		renderCell(bid, 12, true, bidStyle),
		renderCell(ask, 12, true, askStyle),
		renderCell(age, 6, true, ageStyle),
	}, " ")
}

func renderTriangularHeader() string {
	return strings.Join([]string{
		renderCell("", 1, false, plainStyle),
		renderCell("Cycle", 30, false, plainStyle),
		renderCell("Exchange", 10, false, plainStyle),
		renderCell("Start", 8, true, plainStyle),
		renderCell("End", 10, true, plainStyle),
		renderCell("Net %", 9, true, plainStyle),
		renderCell("Fill", 8, true, plainStyle),
		renderCell("Status", 8, true, plainStyle),
	}, " ")
}

func renderTriangularRow(marker string, item arbitrage.TriangularOpportunityV2) string {
	return strings.Join([]string{
		renderCell(marker, 1, false, plainStyle),
		renderCell(truncate(strings.Join(item.Cycle, " -> "), 30), 30, false, plainStyle),
		renderCell(truncate(item.Exchange, 10), 10, false, plainStyle),
		renderCell(item.StartAmount.StringFixed(0), 8, true, plainStyle),
		renderCell(item.EndAmount.StringFixed(2), 10, true, plainStyle),
		renderCell(coloredPercent(item.NetProfitPercent), 9, true, plainStyle),
		renderCell(fillStatus(item.CompleteFill), 8, true, plainStyle),
		renderCell(coloredStatus(item.Status, item.NetProfitPercent), 8, true, plainStyle),
	}, " ")
}

func renderCrossExchangeHeader() string {
	return strings.Join([]string{
		renderCell("", 1, false, plainStyle),
		renderCell("Symbol", 10, false, plainStyle),
		renderCell("Buy On", 10, false, plainStyle),
		renderCell("Buy Avg", 12, true, plainStyle),
		renderCell("Sell On", 10, false, plainStyle),
		renderCell("Sell Avg", 12, true, plainStyle),
		renderCell("Net %", 9, true, plainStyle),
		renderCell("Fill", 8, true, plainStyle),
	}, " ")
}

func renderCrossExchangeRow(marker string, item arbitrage.CrossExchangeOpportunityV2) string {
	return strings.Join([]string{
		renderCell(marker, 1, false, plainStyle),
		renderCell(truncate(item.Symbol, 10), 10, false, plainStyle),
		renderCell(truncate(item.BuyExchange, 10), 10, false, plainStyle),
		renderCell(money(item.BuyAveragePrice), 12, true, plainStyle),
		renderCell(truncate(item.SellExchange, 10), 10, false, plainStyle),
		renderCell(money(item.SellAveragePrice), 12, true, plainStyle),
		renderCell(coloredPercent(item.NetProfitPercent), 9, true, plainStyle),
		renderCell(fillStatus(item.CompleteFill), 8, true, plainStyle),
	}, " ")
}

func renderSpotFuturesHeader() string {
	return strings.Join([]string{
		renderCell("", 1, false, plainStyle),
		renderCell("Symbol", 10, false, plainStyle),
		renderCell("Exchange", 10, false, plainStyle),
		renderCell("Spot Avg", 12, true, plainStyle),
		renderCell("Fut Avg", 12, true, plainStyle),
		renderCell("Basis %", 9, true, plainStyle),
		renderCell("Funding", 9, true, plainStyle),
		renderCell("Fill", 8, true, plainStyle),
	}, " ")
}

func renderSpotFuturesRow(marker string, item arbitrage.SpotFuturesOpportunityV2) string {
	return strings.Join([]string{
		renderCell(marker, 1, false, plainStyle),
		renderCell(truncate(item.Symbol, 10), 10, false, plainStyle),
		renderCell(truncate(item.Exchange, 10), 10, false, plainStyle),
		renderCell(money(item.SpotAverageBuyPrice), 12, true, plainStyle),
		renderCell(money(item.FuturesAverageSellPrice), 12, true, plainStyle),
		renderCell(coloredPercent(item.BasisPercent), 9, true, plainStyle),
		renderCell(coloredPercent(item.FundingRate.Mul(decimal.NewFromInt(100))), 9, true, plainStyle),
		renderCell(fillStatus(item.CompleteFill), 8, true, plainStyle),
	}, " ")
}

func renderCell(text string, width int, alignRight bool, style lipgloss.Style) string {
	if alignRight {
		text = padLeft(text, width)
	} else {
		text = padRight(text, width)
	}
	return style.Render(text)
}

func padLeft(text string, width int) string {
	padding := width - lipgloss.Width(text)
	if padding <= 0 {
		return text
	}
	return strings.Repeat(" ", padding) + text
}

func padRight(text string, width int) string {
	padding := width - lipgloss.Width(text)
	if padding <= 0 {
		return text
	}
	return text + strings.Repeat(" ", padding)
}

func priceHighlightStyle(highlight priceHighlight, fallback lipgloss.Style) lipgloss.Style {
	switch highlight {
	case priceHighlightBest:
		return greenStyle
	case priceHighlightWorst:
		return redStyle
	default:
		return fallback
	}
}

func renderHealthHeader() string {
	return strings.Join([]string{
		renderCell("Exchange", 10, false, plainStyle),
		renderCell("Spot", 7, false, plainStyle),
		renderCell("Futures", 8, false, plainStyle),
		renderCell("WS", 7, false, plainStyle),
		renderCell("REST Fallback", 13, false, plainStyle),
		renderCell("Last Msg", 10, false, plainStyle),
		renderCell("Reconnects", 10, false, plainStyle),
		renderCell("Score", 6, false, plainStyle),
	}, " ")
}

func renderHealthRow(h exchange.ExchangeHealth) string {
	spotText, spotStyle := boolStatusCell(h.SpotEnabled, h.DataFresh)
	futuresText, futuresStyle := boolStatusCell(h.FuturesEnabled, h.DataFresh)
	wsText, wsStyle := wsStatusCell(h)
	fallbackText, fallbackStyle := fallbackStatusCell(h.RestFallbackActive)
	scoreText, scoreStyle := healthScoreCell(h.Score)
	return strings.Join([]string{
		renderCell(truncate(h.Exchange, 10), 10, false, plainStyle),
		renderCell(spotText, 7, false, spotStyle),
		renderCell(futuresText, 8, false, futuresStyle),
		renderCell(wsText, 7, false, wsStyle),
		renderCell(fallbackText, 13, false, fallbackStyle),
		renderCell(ageString(h.LastMessageTime), 10, false, plainStyle),
		renderCell(fmt.Sprintf("%d", h.ReconnectCount), 10, false, plainStyle),
		renderCell(scoreText, 6, false, scoreStyle),
	}, " ")
}

func coloredPercent(value decimal.Decimal) string {
	text := value.StringFixed(3) + "%"
	if value.GreaterThan(decimal.Zero) {
		return greenStyle.Render("+" + text)
	}
	if value.IsNegative() {
		return redStyle.Render(text)
	}
	return mutedStyle.Render(text)
}

func coloredStatus(status string, value decimal.Decimal) string {
	if value.GreaterThan(decimal.Zero) {
		return greenStyle.Render(status)
	}
	if value.IsNegative() {
		return redStyle.Render(status)
	}
	return mutedStyle.Render(status)
}

func fillStatus(complete bool) string {
	if complete {
		return greenStyle.Render("FULL")
	}
	return yellowStyle.Render("PARTIAL")
}

func healthScore(score int) string {
	text, style := healthScoreCell(score)
	return style.Render(text)
}

func healthScoreCell(score int) (string, lipgloss.Style) {
	text := fmt.Sprintf("%d", score)
	switch {
	case score >= 90:
		return text, greenStyle
	case score >= 60:
		return text, yellowStyle
	default:
		return text, redStyle
	}
}

func boolStatus(enabled bool, fresh bool) string {
	text, style := boolStatusCell(enabled, fresh)
	return style.Render(text)
}

func boolStatusCell(enabled bool, fresh bool) (string, lipgloss.Style) {
	if !enabled {
		return "off", mutedStyle
	}
	if fresh {
		return "OK", greenStyle
	}
	return "WARN", yellowStyle
}

func wsStatus(h exchange.ExchangeHealth) string {
	text, style := wsStatusCell(h)
	return style.Render(text)
}

func wsStatusCell(h exchange.ExchangeHealth) (string, lipgloss.Style) {
	if !h.WebSocketEnabled {
		return "off", mutedStyle
	}
	if h.WebSocketConnected {
		return "OK", greenStyle
	}
	return "WARN", yellowStyle
}

func fallbackStatus(active bool) string {
	text, style := fallbackStatusCell(active)
	return style.Render(text)
}

func fallbackStatusCell(active bool) (string, lipgloss.Style) {
	if active {
		return "Yes", yellowStyle
	}
	return "No", greenStyle
}

func (m Model) watchStatus(value decimal.Decimal, complete bool) string {
	if !complete {
		return yellowStyle.Render(m.icons.Partial + " PARTIAL")
	}
	if value.GreaterThan(decimal.Zero) {
		return greenStyle.Render(m.icons.Profit + " WATCH")
	}
	return redStyle.Render(m.icons.Loss + " NO")
}

func (m Model) filterBrokerBasis(items []arbitrage.BrokerFuturesBasisOpportunity) []arbitrage.BrokerFuturesBasisOpportunity {
	if !m.profitableOnly {
		return items
	}
	var out []arbitrage.BrokerFuturesBasisOpportunity
	for _, item := range items {
		if item.NetEstimatePercent.GreaterThan(decimal.Zero) {
			out = append(out, item)
		}
	}
	return out
}

func (m Model) ibkrHealth() exchange.ExchangeHealth {
	for _, h := range m.snapshot.ProviderHealth {
		if strings.EqualFold(h.Provider, "ibkr") || strings.EqualFold(h.Broker, "IBKR") || strings.EqualFold(h.Exchange, "IBKR") {
			return h
		}
	}
	for _, h := range m.snapshot.ExchangeHealth {
		if strings.EqualFold(h.Provider, "ibkr") || strings.EqualFold(h.Broker, "IBKR") || strings.EqualFold(h.Exchange, "IBKR") {
			return h
		}
	}
	return exchange.ExchangeHealth{Provider: "ibkr", Broker: "IBKR", Exchange: "IBKR", Status: "unknown"}
}

func (m Model) ibkrBookLookup() map[string]exchange.OrderBook {
	out := make(map[string]exchange.OrderBook)
	for _, book := range m.snapshot.OrderBooks {
		if !(strings.EqualFold(book.Provider, "ibkr") || strings.EqualFold(book.Broker, "IBKR") || strings.EqualFold(book.Exchange, "IBKR")) {
			continue
		}
		if book.InstrumentID != "" {
			out[book.InstrumentID] = book
		}
	}
	return out
}

func displayMarketName(item exchange.MarketInfo) string {
	if item.DisplayName != "" {
		return item.DisplayName
	}
	if item.InstrumentID != "" {
		return item.InstrumentID
	}
	return item.Symbol
}

func titleOrDefault(titles map[string]string, key string, fallback string) string {
	if titles != nil && titles[key] != "" {
		return titles[key]
	}
	return fallback
}

func ageString(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	age := time.Since(t).Round(time.Second)
	if age < time.Second {
		return "0s"
	}
	return shortDuration(age)
}

func shortDuration(d time.Duration) string {
	if d <= 0 {
		return "n/a"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(time.Second).String()
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "."
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"...") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

func first[S ~[]E, E any](items S, n int) S {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
