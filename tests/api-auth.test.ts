import { describe, it, expect, vi, beforeEach } from 'vitest';
import handler from '../api/auth/[...route]';
import { AuthService } from '../src/lib/auth-service';
import { TwoFactorService } from '../src/lib/two-factor-service';
import { db } from '../src/lib/db';
import { verifyToken } from '../src/lib/auth-middleware';
import { ZodError } from 'zod';

// Mutable state for mock
const mockDbState = {
  queryResult: [] as any[],
};

// Reset mock state before each test
beforeEach(() => {
  mockDbState.queryResult = [];
});

vi.mock('../src/lib/auth-service', () => ({
  AuthService: {
    login: vi.fn(),
    register: vi.fn(),
    verify2FAAndLogin: vi.fn(),
  },
}));

vi.mock('../src/lib/two-factor-service', () => ({
  TwoFactorService: {
    setup: vi.fn(),
    enable: vi.fn(),
  },
}));

vi.mock('../src/lib/db', () => ({
  db: {
    select: vi.fn().mockImplementation(() => ({
      from: vi.fn().mockImplementation(() => ({
        where: vi.fn().mockResolvedValue(mockDbState.queryResult),
      })),
    })),
    insert: vi.fn().mockReturnThis(),
    values: vi.fn().mockReturnThis(),
    returning: vi.fn().mockResolvedValue([]),
  },
}));

vi.mock('../src/lib/auth-middleware', () => ({
  verifyToken: vi.fn(),
}));

vi.mock('../src/lib/rate-limit', () => ({
  rateLimit: vi.fn().mockResolvedValue(true),
}));

describe('Auth Unified Handler', () => {
  let req: any;
  let res: any;

  beforeEach(() => {
    vi.clearAllMocks();
    mockDbState.queryResult = [];
    res = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    };
  });

  it('should return 404 for unknown route', async () => {
    req = { query: { route: ['unknown'] }, method: 'POST' };
    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(404);
  });

  it('should handle login', async () => {
    req = { 
      query: { route: ['login'] }, 
      method: 'POST', 
      body: { email: 'test@example.com', password: 'password' },
      headers: {}
    };
    const mockResult = { access_token: 'jwt', user: { id: 1 } };
    (AuthService.login as any).mockResolvedValue(mockResult);

    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(200);
    expect(res.json).toHaveBeenCalledWith(mockResult);
  });

  it('should handle register', async () => {
    req = { 
      query: { route: ['register'] }, 
      method: 'POST', 
      body: { email: 'test@example.com', password: 'Password123' },
      headers: {}
    };
    const mockUser = { id: 1, email: 'test@example.com' };
    (AuthService.register as any).mockResolvedValue(mockUser);

    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(201);
    expect(res.json).toHaveBeenCalledWith({ message: 'User created successfully', user: mockUser });
  });

  it('should handle /me', async () => {
    req = { query: { route: ['me'] }, method: 'GET', headers: {} };
    (verifyToken as any).mockReturnValue({ userId: 1 });
    mockDbState.queryResult = [{ id: 1, email: 't@e.com', twoFactorEnabled: false }];

    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(200);
  });

  it('should handle 2fa/setup', async () => {
    req = { query: { route: ['2fa', 'setup'] }, method: 'POST', headers: {} };
    (verifyToken as any).mockReturnValue({ userId: 1, email: 't@e.com' });
    (TwoFactorService.setup as any).mockResolvedValue({ secret: 'secret' });

    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(200);
  });

  it('should handle 2fa/enable', async () => {
    req = { 
      query: { route: ['2fa', 'enable'] }, 
      method: 'POST', 
      body: { token: '123456' },
      headers: {} 
    };
    (verifyToken as any).mockReturnValue({ userId: 1 });
    (TwoFactorService.enable as any).mockResolvedValue(undefined);

    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(200);
    expect(res.json).toHaveBeenCalledWith({ message: '2FA enabled successfully' });
  });

  describe('2fa/verify-login', () => {
    it('should succeed with valid userId and token', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: { userId: 1, token: '123456' },
        headers: {} 
      };
      mockDbState.queryResult = [{ id: 1 }];
      const mockResult = { access_token: 'jwt', user: { id: 1 } };
      (AuthService.verify2FAAndLogin as any).mockResolvedValue(mockResult);

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(200);
      expect(res.json).toHaveBeenCalledWith(mockResult);
    });

    it('should return 400 for missing all fields', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: {},
        headers: {} 
      };

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(400);
      expect(res.json).toHaveBeenCalledWith({ error: 'Missing required fields: userId and token' });
    });

    it('should return 400 when only userId is provided', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: { userId: 1 },
        headers: {} 
      };

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(400);
    });

    it('should return 400 when only token is provided', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: { token: '123456' },
        headers: {} 
      };

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(400);
    });

    it('should return 401 for user not found', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: { userId: 999, token: '123456' },
        headers: {} 
      };
      mockDbState.queryResult = [];

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({ code: 'INVALID_CREDENTIALS', message: 'Invalid user or token' });
    });

    it('should return 401 for invalid 2FA token', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: { userId: 1, token: 'wrong' },
        headers: {} 
      };
      mockDbState.queryResult = [{ id: 1 }];
      (AuthService.verify2FAAndLogin as any).mockRejectedValue(new Error('Invalid verification code'));

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({ code: 'INVALID_2FA_TOKEN', message: 'Invalid or expired 2FA token' });
    });

    it('should return 401 when 2FA not enabled for user', async () => {
      req = { 
        query: { route: ['2fa', 'verify-login'] }, 
        method: 'POST', 
        body: { userId: 1, token: '123456' },
        headers: {} 
      };
      mockDbState.queryResult = [{ id: 1 }];
      (AuthService.verify2FAAndLogin as any).mockRejectedValue(new Error('2FA not enabled'));

      await handler(req, res);
      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({ code: 'INVALID_2FA_TOKEN', message: 'Invalid or expired 2FA token' });
    });
  });

  it('should handle ZodError', async () => {
    req = { query: { route: ['login'] }, method: 'POST', body: {}, headers: {} };
    const zodError = new ZodError([{ message: 'Invalid email', path: ['email'], code: 'custom' }]);
    (AuthService.login as any).mockRejectedValue(zodError);

    await handler(req, res);
    expect(res.status).toHaveBeenCalledWith(400);
    expect(res.json).toHaveBeenCalledWith({ error: 'Invalid email' });
  });
});
