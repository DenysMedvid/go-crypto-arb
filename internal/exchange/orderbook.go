package exchange

import (
	"sort"
	"strings"
)

func NormalizeOrderBook(book OrderBook, depth int) OrderBook {
	if book.Provider == "" {
		if book.Exchange != "" {
			book.Provider = strings.ToLower(book.Exchange)
		} else if book.Broker != "" {
			book.Provider = strings.ToLower(book.Broker)
		}
	}
	sort.Slice(book.Bids, func(i, j int) bool {
		return book.Bids[i].Price.GreaterThan(book.Bids[j].Price)
	})
	sort.Slice(book.Asks, func(i, j int) bool {
		return book.Asks[i].Price.LessThan(book.Asks[j].Price)
	})
	if depth > 0 {
		if len(book.Bids) > depth {
			book.Bids = book.Bids[:depth]
		}
		if len(book.Asks) > depth {
			book.Asks = book.Asks[:depth]
		}
	}
	return book
}
