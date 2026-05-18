import { describe, expect, it } from 'vitest';

import { describeApiError } from './apiErrors';

describe('describeApiError', () => {
  it('returns a clear authentication message for 401 responses', () => {
    const result = describeApiError({ status: 401, data: { error: 'invalid API key' } });

    expect(result.authFailed).toBe(true);
    expect(result.message).toContain('invalid API key');
  });

  it('marks fetch failures as backend unavailable', () => {
    const result = describeApiError({ status: 'FETCH_ERROR', error: 'TypeError: Failed to fetch' });

    expect(result.backendUnavailable).toBe(true);
    expect(result.message).toContain('Backend is unavailable');
  });
});
