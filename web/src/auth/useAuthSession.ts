import { useCallback, useEffect, useState } from 'react';
import { getCurrentUser, login as loginRequest } from '../api/auth';
import { ApiError } from '../api/http';
import { clearStoredSession, readStoredSession, writeStoredSession } from './session';
import type { AuthUser } from '../types/auth';

type AuthStatus = 'loading' | 'authenticated' | 'anonymous';

export type AuthSessionState = {
  status: AuthStatus;
  accessToken: string | null;
  expiresAt: string | null;
  user: AuthUser | null;
  errorMessage: string;
  isSubmitting: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
};

export function useAuthSession(): AuthSessionState {
  const [status, setStatus] = useState<AuthStatus>('loading');
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [errorMessage, setErrorMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const resetSession = useCallback(() => {
    clearStoredSession();
    setAccessToken(null);
    setExpiresAt(null);
    setUser(null);
    setStatus('anonymous');
  }, []);

  useEffect(() => {
    let isMounted = true;

    async function bootstrap() {
      const storedSession = readStoredSession();

      if (!storedSession?.accessToken) {
        if (isMounted) {
          setStatus('anonymous');
        }
        return;
      }

      try {
        const currentUser = await getCurrentUser(storedSession.accessToken);

        if (!isMounted) {
          return;
        }

        setAccessToken(storedSession.accessToken);
        setExpiresAt(storedSession.expiresAt);
        setUser(currentUser);
        setStatus('authenticated');
        setErrorMessage('');
        writeStoredSession({
          accessToken: storedSession.accessToken,
          expiresAt: storedSession.expiresAt,
          user: currentUser,
        });
      } catch (error) {
        if (!isMounted) {
          return;
        }

        resetSession();
        setErrorMessage(error instanceof ApiError && error.status === 401 ? '' : 'No se pudo restaurar la sesión.');
      }
    }

    void bootstrap();

    return () => {
      isMounted = false;
    };
  }, [resetSession]);

  const login = useCallback(async (email: string, password: string) => {
    try {
      setIsSubmitting(true);
      setErrorMessage('');

      const response = await loginRequest({ email, password });
      const nextSession = {
        accessToken: response.access_token,
        expiresAt: response.expires_at,
        user: response.user,
      };

      writeStoredSession(nextSession);
      setAccessToken(nextSession.accessToken);
      setExpiresAt(nextSession.expiresAt);
      setUser(nextSession.user);
      setStatus('authenticated');
    } catch (error) {
      resetSession();
      setErrorMessage(getLoginErrorMessage(error));
    } finally {
      setIsSubmitting(false);
    }
  }, [resetSession]);

  const logout = useCallback(() => {
    setErrorMessage('');
    resetSession();
  }, [resetSession]);

  return {
    status,
    accessToken,
    expiresAt,
    user,
    errorMessage,
    isSubmitting,
    login,
    logout,
  };
}

function getLoginErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      return 'Credenciales inválidas. Probá de nuevo.';
    }

    if (error.status === 400) {
      return 'Ingresá email y contraseña.';
    }

    return error.message;
  }

  return 'No se pudo iniciar sesión.';
}
