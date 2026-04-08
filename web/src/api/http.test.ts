import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { clearStoredSession, writeStoredSession } from '../auth/session';
import { request } from './http';

const fetchMock = vi.fn<typeof fetch>();

describe('request auth headers', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    clearStoredSession();
    fetchMock.mockReset();
    vi.unstubAllGlobals();
  });

  it('adds a bearer token when auth is requested and a session exists', async () => {
    writeStoredSession({
      accessToken: 'stored-token',
      expiresAt: '2099-01-01T00:00:00Z',
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        role: 'admin',
        active: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    });

    fetchMock.mockResolvedValue(jsonResponse({ items: [] }));

    await request('/appointments-api', '/appointments', { auth: true });

    expect(fetchMock).toHaveBeenCalledTimes(1);

    const [, init] = fetchMock.mock.calls[0];
    const headers = new Headers(init?.headers);

    expect(headers.get('Authorization')).toBe('Bearer stored-token');
  });

  it('preserves an explicit authorization header instead of overwriting it', async () => {
    writeStoredSession({
      accessToken: 'stored-token',
      expiresAt: '2099-01-01T00:00:00Z',
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        role: 'admin',
        active: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    });

    fetchMock.mockResolvedValue(jsonResponse({ items: [] }));

    await request('/appointments-api', '/appointments', {
      auth: true,
      headers: {
        Authorization: 'Bearer override-token',
      },
    });

    const [, init] = fetchMock.mock.calls[0];
    const headers = new Headers(init?.headers);

    expect(headers.get('Authorization')).toBe('Bearer override-token');
  });

  it('does not add authorization when auth is requested without a stored token', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ items: [] }));

    await request('/appointments-api', '/appointments', { auth: true });

    const [, init] = fetchMock.mock.calls[0];
    const headers = new Headers(init?.headers);

    expect(headers.has('Authorization')).toBe(false);
  });
});

function jsonResponse(payload: unknown) {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: {
      'Content-Type': 'application/json',
    },
  });
}
