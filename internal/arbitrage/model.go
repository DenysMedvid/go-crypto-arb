package arbitrage

import (
	"time"

	"github.com/shopspring/decimal"
)

type TradeSide string

const (
	TradeBuy  TradeSide = "buy"
	TradeSell TradeSide = "sell"
)

type TriangularOpportunity struct {
	Exchange      string          `json:"exchange"`
	Cycle         []string        `json:"cycle"`
	StartAsset    string          `json:"start_asset"`
	StartAmount   decimal.Decimal `json:"start_amount"`
	EndAmount     decimal.Decimal `json:"end_amount"`
	ProfitPercent decimal.Decimal `json:"profit_percent"`
	Status        string          `json:"status"`
	CalculatedAt  time.Time       `json:"calculated_at"`
}

type CrossExchangeOpportunity struct {
	Symbol       string          `json:"symbol"`
	BuyExchange  string          `json:"buy_exchange"`
	SellExchange string          `json:"sell_exchange"`
	BuyAsk       decimal.Decimal `json:"buy_ask"`
	SellBid      decimal.Decimal `json:"sell_bid"`
	BuyFee       decimal.Decimal `json:"buy_fee"`
	SellFee      decimal.Decimal `json:"sell_fee"`
	NetPercent   decimal.Decimal `json:"net_percent"`
	Status       string          `json:"status"`
	CalculatedAt time.Time       `json:"calculated_at"`
}

type SpotFuturesOpportunity struct {
	Symbol       string          `json:"symbol"`
	Exchange     string          `json:"exchange"`
	SpotAsk      decimal.Decimal `json:"spot_ask"`
	FuturesBid   decimal.Decimal `json:"futures_bid"`
	BasisPercent decimal.Decimal `json:"basis_percent"`
	FundingRate  decimal.Decimal `json:"funding_rate"`
	NetEstimate  decimal.Decimal `json:"net_estimate"`
	Status       string          `json:"status"`
	CalculatedAt time.Time       `json:"calculated_at"`
}

type ExecutionSimulation struct {
	Symbol              string          `json:"symbol"`
	Side                TradeSide       `json:"side"`
	RequestedQuoteValue decimal.Decimal `json:"requested_quote_value"`
	RequestedBaseQty    decimal.Decimal `json:"requested_base_qty"`
	FilledBaseQty       decimal.Decimal `json:"filled_base_qty"`
	SpentQuoteValue     decimal.Decimal `json:"spent_quote_value"`
	ReceivedQuoteValue  decimal.Decimal `json:"received_quote_value"`
	AveragePrice        decimal.Decimal `json:"average_price"`
	BestPrice           decimal.Decimal `json:"best_price"`
	SlippagePercent     decimal.Decimal `json:"slippage_percent"`
	FeePercent          decimal.Decimal `json:"fee_percent"`
	FeeAmount           decimal.Decimal `json:"fee_amount"`
	CompleteFill        bool            `json:"complete_fill"`
	LimitedDepth        bool            `json:"limited_depth"`
	Error               string          `json:"error,omitempty"`
	Status              string          `json:"status,omitempty"`
}

type LegSimulation struct {
	FromAsset       string          `json:"from_asset"`
	ToAsset         string          `json:"to_asset"`
	Symbol          string          `json:"symbol"`
	Side            TradeSide       `json:"side"`
	InputAmount     decimal.Decimal `json:"input_amount"`
	OutputAmount    decimal.Decimal `json:"output_amount"`
	AveragePrice    decimal.Decimal `json:"average_price"`
	FeeAmount       decimal.Decimal `json:"fee_amount"`
	SlippagePercent decimal.Decimal `json:"slippage_percent"`
	CompleteFill    bool            `json:"complete_fill"`
}

type TriangularOpportunityV2 struct {
	Provider           string          `json:"provider"`
	Exchange           string          `json:"exchange"`
	StrategyTitle      string          `json:"strategy_title"`
	AssetClass         string          `json:"asset_class,omitempty"`
	Cycle              []string        `json:"cycle"`
	StartAsset         string          `json:"start_asset"`
	StartAmount        decimal.Decimal `json:"start_amount"`
	EndAmount          decimal.Decimal `json:"end_amount"`
	NetProfitPercent   decimal.Decimal `json:"net_profit_percent"`
	CompleteFill       bool            `json:"complete_fill"`
	WorstLeg           string          `json:"worst_leg"`
	MaxSlippagePercent decimal.Decimal `json:"max_slippage_percent"`
	Legs               []LegSimulation `json:"legs"`
	Status             string          `json:"status"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type CrossExchangeOpportunityV2 struct {
	StrategyTitle       string          `json:"strategy_title"`
	Symbol              string          `json:"symbol"`
	BuyProvider         string          `json:"buy_provider"`
	SellProvider        string          `json:"sell_provider"`
	BuyExchange         string          `json:"buy_exchange"`
	SellExchange        string          `json:"sell_exchange"`
	TradeSize           decimal.Decimal `json:"trade_size"`
	BuyAveragePrice     decimal.Decimal `json:"buy_average_price"`
	SellAveragePrice    decimal.Decimal `json:"sell_average_price"`
	BuySlippagePercent  decimal.Decimal `json:"buy_slippage_percent"`
	SellSlippagePercent decimal.Decimal `json:"sell_slippage_percent"`
	BuyFeeAmount        decimal.Decimal `json:"buy_fee_amount"`
	SellFeeAmount       decimal.Decimal `json:"sell_fee_amount"`
	NetProfitPercent    decimal.Decimal `json:"net_profit_percent"`
	CompleteFill        bool            `json:"complete_fill"`
	Status              string          `json:"status"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type SpotFuturesOpportunityV2 struct {
	StrategyTitle           string          `json:"strategy_title"`
	Provider                string          `json:"provider"`
	Exchange                string          `json:"exchange"`
	Symbol                  string          `json:"symbol"`
	TradeSize               decimal.Decimal `json:"trade_size"`
	SpotAverageBuyPrice     decimal.Decimal `json:"spot_average_buy_price"`
	FuturesAverageSellPrice decimal.Decimal `json:"futures_average_sell_price"`
	SpotSlippagePercent     decimal.Decimal `json:"spot_slippage_percent"`
	FuturesSlippagePercent  decimal.Decimal `json:"futures_slippage_percent"`
	SpotFeeAmount           decimal.Decimal `json:"spot_fee_amount"`
	FuturesFeeAmount        decimal.Decimal `json:"futures_fee_amount"`
	BasisPercent            decimal.Decimal `json:"basis_percent"`
	FundingRate             decimal.Decimal `json:"funding_rate"`
	NetEstimatePercent      decimal.Decimal `json:"net_estimate_percent"`
	CompleteFill            bool            `json:"complete_fill"`
	Status                  string          `json:"status"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

type BrokerFuturesBasisOpportunity struct {
	StrategyTitle       string          `json:"strategy_title"`
	Asset               string          `json:"asset"`
	SpotProvider        string          `json:"spot_provider"`
	SpotSymbol          string          `json:"spot_symbol"`
	SpotAsk             decimal.Decimal `json:"spot_ask"`
	FuturesProvider     string          `json:"futures_provider"`
	FuturesInstrumentID string          `json:"futures_instrument_id"`
	FuturesDisplayName  string          `json:"futures_display_name"`
	FuturesBid          decimal.Decimal `json:"futures_bid"`
	BasisPercent        decimal.Decimal `json:"basis_percent"`
	NetEstimatePercent  decimal.Decimal `json:"net_estimate_percent"`
	CompleteFill        bool            `json:"complete_fill"`
	Status              string          `json:"status"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type RelatedAssetGroupSignal struct {
	Group        string               `json:"group"`
	Assets       []RelatedAssetSignal `json:"assets"`
	GroupAverage decimal.Decimal      `json:"group_average"`
	CalculatedAt time.Time            `json:"calculated_at"`
}

type RelatedAssetSignal struct {
	Symbol            string          `json:"symbol"`
	Asset             string          `json:"asset"`
	Exchange          string          `json:"exchange"`
	ChangePercent     decimal.Decimal `json:"change_percent"`
	DivergencePercent decimal.Decimal `json:"divergence_percent"`
}
