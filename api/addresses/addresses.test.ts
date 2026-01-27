import { describe, it, expect, vi, beforeEach } from 'vitest';

// Must set env before importing handler
const originalEnv = { ...process.env };

describe('/api/addresses', () => {
  beforeEach(() => {
    // Reset environment to original state before each test
    process.env = { ...originalEnv };
    vi.clearAllMocks();
  });

  describe('index.ts (GET/POST)', () => {
    it('should return 405 for DELETE requests (handled by index.ts)', async () => {
      // Re-import handler to pick up env changes if any (though we reset env)
      const handler = await import('./index.js').then(m => m.default);
      
      const req = {
        method: 'DELETE',
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

    it('should proxy POST request to backend', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./index.js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: vi.fn().mockResolvedValue({ id: 1, wallet_address: '0x123' }),
      });

      const req = {
        method: 'POST',
        headers: { authorization: 'Bearer token' },
        body: { wallet_address: '0x123', chain_type: 'ETH' },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(201);
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/addresses',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Authorization': 'Bearer token',
          }),
          body: JSON.stringify({ wallet_address: '0x123', chain_type: 'ETH' }),
        })
      );
    });

    it('should proxy GET request to backend', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./index.js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ addresses: [] }),
      });

      const req = {
        method: 'GET',
        headers: { authorization: 'Bearer token' },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(200);
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/addresses',
        expect.objectContaining({
          method: 'GET',
        })
      );
    });
  });

  describe('[...path].ts (Sub-resources)', () => {
    it('should proxy DELETE request with ID', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./[...path].js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ message: 'Deleted' }),
      });

      const req = {
        method: 'DELETE',
        query: { path: ['123'] },
        headers: { authorization: 'Bearer token' },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(200);
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/addresses/123',
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });

    it('should proxy POST verify request', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./[...path].js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ message: 'Verified' }),
      });

      const req = {
        method: 'POST',
        query: { path: ['123', 'verify'] },
        headers: { authorization: 'Bearer token' },
        body: { token: '123456' },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(200);
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/addresses/123/verify',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ token: '123456' }),
        })
      );
    });
  });
});
