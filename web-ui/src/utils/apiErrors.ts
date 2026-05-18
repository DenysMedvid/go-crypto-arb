import type { FetchBaseQueryError } from '@reduxjs/toolkit/query';

export interface ApiErrorView {
  message: string;
  authFailed: boolean;
  backendUnavailable: boolean;
  status?: string;
}

export function describeApiError(error: FetchBaseQueryError): ApiErrorView {
  if (typeof error.status === 'number') {
    const bodyMessage = extractErrorMessage(error.data);
    if (error.status === 401) {
      return {
        message: bodyMessage || 'API authentication failed. Check the configured API key.',
        authFailed: true,
        backendUnavailable: false,
        status: String(error.status),
      };
    }
    return {
      message: bodyMessage || `Backend returned HTTP ${error.status}.`,
      authFailed: false,
      backendUnavailable: false,
      status: String(error.status),
    };
  }

  if (error.status === 'FETCH_ERROR') {
    return {
      message: 'Backend is unavailable or blocked by the browser. Check the API base URL.',
      authFailed: false,
      backendUnavailable: true,
      status: error.status,
    };
  }

  if (error.status === 'PARSING_ERROR') {
    return {
      message: 'Backend response could not be parsed as the expected format.',
      authFailed: false,
      backendUnavailable: false,
      status: error.status,
    };
  }

  return {
    message: 'Request failed before reaching the backend.',
    authFailed: false,
    backendUnavailable: false,
    status: String(error.status),
  };
}

function extractErrorMessage(data: unknown): string | undefined {
  if (typeof data === 'object' && data !== null && 'error' in data) {
    const maybeError = (data as { error?: unknown }).error;
    return typeof maybeError === 'string' ? maybeError : undefined;
  }
  return undefined;
}
