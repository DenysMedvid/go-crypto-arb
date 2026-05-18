package exchange

import (
	"sort"
	"strings"
)

func CanonicalSymbol(base, quote string) string {
	base = NormalizeAsset(base)
	quote = NormalizeAsset(quote)
	if base == "" || quote == "" {
		return ""
	}
	return base + "/" + quote
}

func NormalizeAsset(asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	switch asset {
	case "XBT":
		return "BTC"
	default:
		return asset
	}
}

func NormalizeCanonicalSymbol(symbol string) string {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	symbol = strings.ReplaceAll(symbol, "-", "/")
	if strings.Contains(symbol, ":") {
		parts := strings.Split(symbol, ":")
		if len(parts) == 2 {
			return CanonicalSymbol(parts[0], parts[1])
		}
	}
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return symbol
	}
	return CanonicalSymbol(parts[0], parts[1])
}

func SplitJoinedSymbol(symbol string, knownAssets []string) (base, quote, canonical string, ok bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	assets := normalizeUnique(knownAssets)
	sort.SliceStable(assets, func(i, j int) bool {
		return len(assets[i]) > len(assets[j])
	})
	for _, q := range assets {
		if !strings.HasSuffix(symbol, q) {
			continue
		}
		b := strings.TrimSuffix(symbol, q)
		if b == "" || b == q {
			continue
		}
		b = NormalizeAsset(b)
		q = NormalizeAsset(q)
		if contains(assets, b) {
			return b, q, CanonicalSymbol(b, q), true
		}
	}
	return "", "", "", false
}

func JoinedSymbol(symbol string) string {
	parts := strings.Split(NormalizeCanonicalSymbol(symbol), "/")
	if len(parts) != 2 {
		return strings.ReplaceAll(strings.ToUpper(symbol), "/", "")
	}
	return parts[0] + parts[1]
}

func normalizeUnique(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, item := range in {
		item = NormalizeAsset(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func contains(in []string, value string) bool {
	value = NormalizeAsset(value)
	for _, item := range in {
		if NormalizeAsset(item) == value {
			return true
		}
	}
	return false
}
