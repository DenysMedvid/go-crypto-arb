import type { Decimal } from '../api/types';

export function decimalToText(value: Decimal | null | undefined): string {
  if (value === null || value === undefined || value === '') {
    return 'n/a';
  }
  return String(value);
}

export function decimalToNumber(value: Decimal | null | undefined): number {
  if (value === null || value === undefined || value === '') {
    return 0;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function formatDecimal(value: Decimal | null | undefined, digits = 4): string {
  const text = decimalToText(value);
  if (text === 'n/a') {
    return text;
  }
  const parsed = Number(text);
  if (!Number.isFinite(parsed)) {
    return text;
  }
  if (Math.abs(parsed) >= 1000) {
    return parsed.toLocaleString(undefined, {
      maximumFractionDigits: 2,
      minimumFractionDigits: 2,
    });
  }
  return parsed.toLocaleString(undefined, {
    maximumFractionDigits: digits,
    minimumFractionDigits: Math.min(2, digits),
  });
}

export function formatPercent(value: Decimal | null | undefined, digits = 3): string {
  const parsed = decimalToNumber(value);
  const sign = parsed > 0 ? '+' : '';
  return `${sign}${parsed.toFixed(digits)}%`;
}

export function profitClass(value: Decimal | null | undefined): 'positive' | 'negative' | 'muted' {
  const parsed = decimalToNumber(value);
  if (parsed > 0) {
    return 'positive';
  }
  if (parsed < 0) {
    return 'negative';
  }
  return 'muted';
}
