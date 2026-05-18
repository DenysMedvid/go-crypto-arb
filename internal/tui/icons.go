package tui

type IconSet struct {
	App            string
	Prices         string
	Triangular     string
	CrossExchange  string
	SpotFutures    string
	Signals        string
	Alerts         string
	Health         string
	IBKR           string
	OK             string
	Warning        string
	Error          string
	Profit         string
	Loss           string
	Partial        string
	Locked         string
	MarketDataOnly string
}

func NewIconSet(useEmoji bool) IconSet {
	if useEmoji {
		return IconSet{
			App:            "📊",
			Prices:         "💰",
			Triangular:     "🔺",
			CrossExchange:  "🔁",
			SpotFutures:    "📈",
			Signals:        "🧭",
			Alerts:         "🚨",
			Health:         "🩺",
			IBKR:           "🏦",
			OK:             "✅",
			Warning:        "⚠️",
			Error:          "❌",
			Profit:         "🟢",
			Loss:           "🔴",
			Partial:        "🟡",
			Locked:         "🔒",
			MarketDataOnly: "👁",
		}
	}
	return IconSet{
		App:            "[APP]",
		Prices:         "$",
		Triangular:     "^",
		CrossExchange:  "<>",
		SpotFutures:    "%",
		Signals:        "~",
		Alerts:         "!",
		Health:         "+",
		IBKR:           "IBKR",
		OK:             "OK",
		Warning:        "WARN",
		Error:          "ERR",
		Profit:         "+",
		Loss:           "-",
		Partial:        "~",
		Locked:         "LOCK",
		MarketDataOnly: "VIEW",
	}
}
