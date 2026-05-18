package config

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

type Decimal struct {
	decimal.Decimal
}

func (d *Decimal) UnmarshalYAML(value *yaml.Node) error {
	var raw any
	if err := value.Decode(&raw); err != nil {
		return err
	}
	parsed, err := decimal.NewFromString(fmt.Sprint(raw))
	if err != nil {
		return fmt.Errorf("parse decimal %q: %w", fmt.Sprint(raw), err)
	}
	d.Decimal = parsed
	return nil
}

func (d Decimal) DecimalValue() decimal.Decimal {
	return d.Decimal
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return d.Decimal.MarshalJSON()
}

func MustDecimal(value string) Decimal {
	return Decimal{Decimal: decimal.RequireFromString(value)}
}
