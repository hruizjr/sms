import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import {
  ApiError,
  createApiClient,
  type ApiClient,
  type Permission,
  type Session,
  type Transport,
} from '../../lib/api/client';

/*
 * The signed-in session (owner decisions G3-1 to G3-6). The browser keeps nothing but what is in
 * memory here: the session itself is an HttpOnly cookie managed by the server. A 401 from any request
 * (expired, signed out elsewhere, revoked) returns the portal to the sign-in page centrally.
 */

export type SessionState =
  | { status: 'loading' }
  | { status: 'unavailable' }
  | { status: 'signed-out'; notice?: string }
  | { status: 'signed-in'; session: Session };

type SessionContextValue = {
  state: SessionState;
  api: ApiClient;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  retry: () => void;
};

const SessionContext = createContext<SessionContextValue | null>(null);

export const SESSION_ENDED_NOTICE = 'Your session has ended. Please sign in again.';

/**
 * Provides the session and the API client. The transport is fixed for the provider's lifetime.
 */
export function SessionProvider({
  transport,
  children,
}: {
  transport: Transport;
  children: ReactNode;
}) {
  const [state, setState] = useState<SessionState>({ status: 'loading' });
  const [attempt, setAttempt] = useState(0);

  // The CSRF token lives in this closure (memory only), shared by the client and the callbacks.
  const [{ api, setCsrf }] = useState(() => {
    let token: string | undefined;
    const client = createApiClient({
      transport,
      csrfToken: () => token,
      onUnauthenticated: () => {
        token = undefined;
        setState((current) =>
          current.status === 'signed-in'
            ? { status: 'signed-out', notice: SESSION_ENDED_NOTICE }
            : current,
        );
      },
    });
    return {
      api: client,
      setCsrf: (value: string | undefined) => {
        token = value;
      },
    };
  });

  useEffect(() => {
    let active = true;
    api.getSession().then(
      (session) => {
        if (!active) return;
        setCsrf(session.csrf_token);
        setState({ status: 'signed-in', session });
      },
      (error: unknown) => {
        if (!active) return;
        setState(
          error instanceof ApiError && (error.status === 401 || error.status === 403)
            ? { status: 'signed-out' }
            : { status: 'unavailable' },
        );
      },
    );
    return () => {
      active = false;
    };
  }, [api, setCsrf, attempt]);

  const signIn = useCallback(
    async (email: string, password: string) => {
      const session = await api.signIn(email, password);
      setCsrf(session.csrf_token);
      setState({ status: 'signed-in', session });
    },
    [api, setCsrf],
  );

  const signOut = useCallback(async () => {
    try {
      await api.signOut();
    } finally {
      setCsrf(undefined);
      setState({ status: 'signed-out' });
    }
  }, [api, setCsrf]);

  const retry = useCallback(() => {
    setState({ status: 'loading' });
    setAttempt((n) => n + 1);
  }, []);

  const value = useMemo(
    () => ({ state, api, signIn, signOut, retry }),
    [state, api, signIn, signOut, retry],
  );
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  const value = useContext(SessionContext);
  if (value === null) {
    throw new Error('useSession must be used inside <SessionProvider>');
  }
  return value;
}

/** The signed-in session; for components rendered only when signed in. */
export function useSignedInSession(): Session {
  const { state } = useSession();
  if (state.status !== 'signed-in') {
    throw new Error('useSignedInSession used without a signed-in session');
  }
  return state.session;
}

export function hasPermission(session: Session, permission: Permission): boolean {
  return session.permissions.includes(permission);
}
