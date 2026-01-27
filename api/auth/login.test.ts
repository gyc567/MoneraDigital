import { describe, it, expect, vi, beforeEach } from 'vitest';

// Must set env before importing handler
const originalEnv = { ...process.env };

describe('/api/auth/login', () => {
  beforeEach(() => {
    // Reset environment to original state before each test
    process.env = { ...originalEnv };
    vi.clearAllMocks();
  });

  it('should return 405 for non-POST requests', async () => {
    const handler = await import('./login.js').then(m => m.default);
    const req = {
      method: 'GET',
      body: {},
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(405);
    expect(res.json).toHaveBeenCalledWith({ error: 'Method Not Allowed' });
  });

  it('should return 500 when BACKEND_URL is not configured', async () => {
    // Mock environment without BACKEND_URL
    process.env.BACKEND_URL = undefined;

    const handler = await import('./login.js').then(m => m.default);

    const req = {
      method: 'POST',
      body: { email: 'test@example.com', password: 'password' },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(500);
    expect(res.json).toHaveBeenCalledWith({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.',
    });
  });

  it('should proxy valid login request to configured backend', async () => {
    // Mock environment with valid backend URL
    process.env.BACKEND_URL = 'https://monera-digital--gyc567.replit.app';

    const handler = await import('./login.js').then(m => m.default);

    // Mock fetch
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: vi.fn().mockResolvedValue({
        token: 'test-token',
        user: { id: 1, email: 'test@example.com' },
      }),
    });

    const req = {
      method: 'POST',
      body: { email: 'test@example.com', password: 'password' },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(200);
    expect(global.fetch).toHaveBeenCalledWith(
      'https://monera-digital--gyc567.replit.app/api/auth/login',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'test@example.com', password: 'password' }),
      })
    );
  });

  it('should proxy authentication error from backend', async () => {
    // Mock environment with valid backend URL
    process.env.BACKEND_URL = 'https://monera-digital--gyc567.replit.app';

    const handler = await import('./login.js').then(m => m.default);

    // Mock fetch with 401 response
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: vi.fn().mockResolvedValue({
        error: 'invalid credentials',
      }),
    });

    const req = {
      method: 'POST',
      body: { email: 'test@example.com', password: 'wrongpassword' },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith({
      error: 'invalid credentials',
    });
  });

  it('should return 500 when backend connection fails', async () => {
    // Mock environment with valid backend URL
    process.env.BACKEND_URL = 'https://monera-digital--gyc567.replit.app';

    const handler = await import('./login.js').then(m => m.default);

    // Mock fetch with network error
    global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

    const req = {
      method: 'POST',
      body: { email: 'test@example.com', password: 'password' },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(500);
    expect(res.json).toHaveBeenCalledWith({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  });
});
