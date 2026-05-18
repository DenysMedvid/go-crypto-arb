export const DEFAULT_STALE_AFTER_MS = 15_000;

export function ageMs(updatedAt: string | undefined, now = Date.now()): number | undefined {
  if (!updatedAt) {
    return undefined;
  }
  const parsed = Date.parse(updatedAt);
  if (!Number.isFinite(parsed)) {
    return undefined;
  }
  return Math.max(0, now - parsed);
}

export function isStale(updatedAt: string | undefined, staleAfterMs = DEFAULT_STALE_AFTER_MS): boolean {
  const age = ageMs(updatedAt);
  return age === undefined ? true : age > staleAfterMs;
}

export function formatAge(updatedAt: string | undefined, now = Date.now()): string {
  const age = ageMs(updatedAt, now);
  if (age === undefined) {
    return 'n/a';
  }
  if (age < 1000) {
    return '0s';
  }
  const seconds = Math.round(age / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  if (minutes < 60) {
    return rest === 0 ? `${minutes}m` : `${minutes}m ${rest}s`;
  }
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
}

export function formatDateTime(value: string | undefined): string {
  if (!value) {
    return 'never';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}
