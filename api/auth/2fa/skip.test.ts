import { describe, it, expect, vi, beforeEach } from 'vitest';

// Set JWT secret before importing handlers
process.env.JWT_SECRET = 'test-secret-key-for-jwt-verification-minimum-32-bytes';

const originalEnv = { ...process.env };

describe('/api/auth/2fa/skip', () => {
  beforeEach(() => {
    process.env = { ...originalEnv };
    vi.clearAllMocks();
  });

  it('should return 405 for non-POST requests', async () => {
    process.env.BACKEND_URL = 'http://localhost:8081';
    const handler = await import('./skip.js').then(m => m.default);

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

  it('should return 400 when userId is missing', async () => {
    process.env.BACKEND_URL = 'http://localhost:8081';
    const handler = await import('./skip.js').then(m => m.default);

    const req = {
      method: 'POST',
      body: {},
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(400);
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({
        error: 'Invalid request',
      })
    );
  });

  it('should proxy skip request to backend successfully', async () => {
    process.env.BACKEND_URL = 'http://localhost:8081';
    const handler = await import('./skip.js').then(m => m.default);

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: vi.fn().mockResolvedValue({
        token: 'test-jwt-token',
        user: { id: 1, email: 'test@example.com' },
        message: '2FA skipped successfully',
      }),
    });

    const req = {
      method: 'POST',
      body: { userId: 1 },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(200);
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8081/api/auth/2fa/skip',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ userId: 1 }),
      })
    );
  });

  it('should handle backend error response', async () => {
    process.env.BACKEND_URL = 'http://localhost:8081';
    const handler = await import('./skip.js').then(m => m.default);

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: vi.fn().mockResolvedValue({
        error: 'Invalid user',
        message: 'User not found',
      }),
    });

    const req = {
      method: 'POST',
      body: { userId: 999 },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(400);
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({
        error: 'Invalid user',
      })
    );
  });

  it('should handle network errors', async () => {
    process.env.BACKEND_URL = 'http://localhost:8081';
    const handler = await import('./skip.js').then(m => m.default);

    global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

    const req = {
      method: 'POST',
      body: { userId: 1 },
    } as any;

    const res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    } as any;

    await handler(req, res);

    expect(res.status).toHaveBeenCalledWith(500);
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({
        error: 'Internal Server Error',
      })
    );
  });
});
