import { ApiError, NetworkError, type ApiKeyScope } from '../../lib/api/client';

/** What each public API permission allows (contract D-143). */
export const SCOPE_LABELS: Record<ApiKeyScope, { title: string; description: string }> = {
  'messages:send': {
    title: 'Send messages',
    description: 'Submit SMS with POST /v1/messages.',
  },
  'messages:read': {
    title: 'Read messages',
    description: 'Read message status with GET /v1/messages.',
  },
};

export const SCOPES = Object.keys(SCOPE_LABELS) as ApiKeyScope[];

/** A user-facing message for a failed API key request. */
export function apiKeyErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof NetworkError) {
    return 'The server could not be reached. Check your connection and try again.';
  }
  if (error instanceof ApiError && error.problemType === 'forbidden') {
    return 'Your role does not allow managing API keys.';
  }
  if (error instanceof ApiError && error.problemType === 'service-busy') {
    return 'The service is busy and could not check your password. Nothing was created. Try again in a moment.';
  }
  return fallback;
}
