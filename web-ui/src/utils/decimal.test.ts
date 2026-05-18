import { describe, expect, it } from 'vitest';

import { decimalToNumber, decimalToText, formatDecimal, formatPercent, profitClass } from './decimal';

describe('decimal utilities', () => {
  it('keeps API decimal values printable without mutating string precision', () => {
    expect(decimalToText('67210.20000001')).toBe('67210.20000001');
    expect(decimalToNumber('0.125')).toBe(0.125);
  });

  it('formats decimals and percents for tables', () => {
    expect(formatDecimal('67210.2')).toBe('67,210.20');
    expect(formatPercent('0.12345')).toBe('+0.123%');
  });

  it('classifies profit tone', () => {
    expect(profitClass('1')).toBe('positive');
    expect(profitClass('-0.1')).toBe('negative');
    expect(profitClass('0')).toBe('muted');
  });
});
