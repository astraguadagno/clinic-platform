import { ApiError } from '../api/http';

export type AuthenticatedViewResolution =
  | { kind: 'session-invalid' }
  | { kind: 'forbidden'; message: string }
  | { kind: 'error'; message: string };

export function resolveAuthenticatedViewError(
  error: unknown,
  onSessionInvalid: () => void,
  fallbackMessage: string,
  forbiddenFallbackMessage = 'Acceso denegado.',
): AuthenticatedViewResolution {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      onSessionInvalid();
      return { kind: 'session-invalid' };
    }

    if (error.status === 403) {
      return {
        kind: 'forbidden',
        message: error.message || forbiddenFallbackMessage,
      };
    }

    return { kind: 'error', message: error.message || fallbackMessage };
  }

  if (error instanceof Error) {
    return { kind: 'error', message: error.message };
  }

  return { kind: 'error', message: fallbackMessage };
}
