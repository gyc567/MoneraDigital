import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { VercelRequest, VercelResponse } from '@vercel/node';
import handler from '../api/auth/login.js';

// Mock the auth service
vi.mock('../src/lib/auth-service.js', () => ({
  AuthService: {
    login: vi.fn(),
  },
}));

// Mock the logger
vi.mock('../src/lib/logger.js', () => ({
  default: {
    error: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
  },
}));

describe('POST /api/auth/login - Fix for 500 Error', () => {
  let mockReq: Partial<VercelRequest>;
  let mockRes: Partial<VercelResponse>;
  let statusMock: any;
  let jsonMock: any;

  beforeEach(() => {
    // Reset mocks
    vi.clearAllMocks();

    // Setup response mocks
    jsonMock = vi.fn().mockReturnValue(undefined);
    statusMock = vi.fn().mockReturnValue({ json: jsonMock });

    mockReq = {
      method: 'POST',
      body: {},
    };

    mockRes = {
      status: statusMock,
      json: jsonMock,
    } as any;
  });

  it('should return 200 with token on successful login without 2FA', async () => {
    const { AuthService } = await import('../src/lib/auth-service.js');

    mockReq.body = { email: 'user@example.com', password: 'password123' };

    vi.mocked(AuthService.login).mockResolvedValue({
      requires2FA: false,
      user: { id: 1, email: 'user@example.com' },
      token: 'jwt-token-xyz',
    } as any);

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    // Verify status 200 was called
    expect(statusMock).toHaveBeenCalledWith(200);

    // Verify json was called with correct response
    expect(jsonMock).toHaveBeenCalledWith({
      user: { id: 1, email: 'user@example.com' },
      token: 'jwt-token-xyz',
    });

    // Verify only one response was sent (critical fix verification)
    expect(statusMock).toHaveBeenCalledTimes(1);
    expect(jsonMock).toHaveBeenCalledTimes(1);
  });

  it('should return 200 with userId for 2FA-enabled users', async () => {
    const { AuthService } = await import('../src/lib/auth-service.js');

    mockReq.body = { email: '2fa@example.com', password: 'password123' };

    vi.mocked(AuthService.login).mockResolvedValue({
      requires2FA: true,
      userId: 42,
    } as any);

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(200);
    expect(jsonMock).toHaveBeenCalledWith({
      requires2FA: true,
      userId: 42,
    });

    // Verify only one response sent
    expect(statusMock).toHaveBeenCalledTimes(1);
    expect(jsonMock).toHaveBeenCalledTimes(1);
  });

  it('should return 400 when email is missing', async () => {
    mockReq.body = { password: 'password123' };

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(400);
    expect(jsonMock).toHaveBeenCalledWith({
      error: 'Email and password are required',
    });
  });

  it('should return 400 when password is missing', async () => {
    mockReq.body = { email: 'user@example.com' };

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(400);
    expect(jsonMock).toHaveBeenCalledWith({
      error: 'Email and password are required',
    });
  });

  it('should return 401 on invalid credentials', async () => {
    const { AuthService } = await import('../src/lib/auth-service.js');

    mockReq.body = { email: 'user@example.com', password: 'wrongpassword' };

    vi.mocked(AuthService.login).mockRejectedValue(
      new Error('Invalid email or password')
    );

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(401);
    expect(jsonMock).toHaveBeenCalledWith({
      code: 'INVALID_CREDENTIALS',
      error: 'Invalid email or password',
    });
  });

  it('should return 405 on non-POST request', async () => {
    mockReq.method = 'GET';

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(405);
    expect(jsonMock).toHaveBeenCalledWith({
      error: 'Method Not Allowed',
    });
  });

  it('should return 500 with generic error on unexpected database error', async () => {
    const { AuthService } = await import('../src/lib/auth-service.js');

    mockReq.body = { email: 'user@example.com', password: 'password123' };

    vi.mocked(AuthService.login).mockRejectedValue(
      new Error('Database connection failed')
    );

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(500);
    expect(jsonMock).toHaveBeenCalledWith({
      error: 'Internal Server Error',
    });
  });

  it('should handle non-Error objects thrown as errors', async () => {
    const { AuthService } = await import('../src/lib/auth-service.js');

    mockReq.body = { email: 'user@example.com', password: 'password123' };

    vi.mocked(AuthService.login).mockRejectedValue('Unknown error string');

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    expect(statusMock).toHaveBeenCalledWith(500);
    expect(jsonMock).toHaveBeenCalledWith({
      error: 'Internal Server Error',
    });
  });

  it('should log errors with stack trace for debugging', async () => {
    const { AuthService } = await import('../src/lib/auth-service.js');
    const logger = await import('../src/lib/logger.js');

    mockReq.body = { email: 'user@example.com', password: 'password123' };

    const testError = new Error('Test database error');
    vi.mocked(AuthService.login).mockRejectedValue(testError);

    await handler(mockReq as VercelRequest, mockRes as VercelResponse);

    // Verify error was logged with stack trace
    expect(logger.default.error).toHaveBeenCalled();
    const logCall = vi.mocked(logger.default.error).mock.calls[0];
    expect(logCall[0]).toMatchObject({
      error: 'Test database error',
      stack: expect.any(String),
    });
  });
});
