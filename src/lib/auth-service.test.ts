import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AuthService } from './auth-service';
import sql from './db';
import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';

// Mock dependencies
vi.mock('./db', () => ({
  default: vi.fn(),
}));

vi.mock('bcryptjs', () => ({
  default: {
    hash: vi.fn(),
    compare: vi.fn(),
  },
}));

vi.mock('jsonwebtoken', () => ({
  default: {
    sign: vi.fn(),
  },
}));

describe('AuthService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('register', () => {
    it('should successfully register a user', async () => {
      const mockUser = { id: 1, email: 'test@example.com' };
      (bcrypt.hash as any).mockResolvedValue('hashed_password');
      (sql as any).mockResolvedValue([mockUser]);

      const result = await AuthService.register('test@example.com', 'password123');

      expect(result).toEqual(mockUser);
      expect(bcrypt.hash).toHaveBeenCalledWith('password123', 10);
    });

    it('should throw error if email or password missing', async () => {
      await expect(AuthService.register('', '')).rejects.toThrow('Email and password are required');
    });

    it('should throw error if user already exists', async () => {
      (bcrypt.hash as any).mockResolvedValue('hashed_password');
      (sql as any).mockRejectedValue({ code: '23505' });

      await expect(AuthService.register('test@example.com', 'password123')).rejects.toThrow('User already exists');
    });

    it('should rethrow unknown errors', async () => {
      (sql as any).mockRejectedValue(new Error('DB Error'));
      await expect(AuthService.register('test@example.com', 'password123')).rejects.toThrow('DB Error');
    });
  });

  describe('login', () => {
    it('should successfully login a user and return a token', async () => {
      const mockUser = { id: 1, email: 'test@example.com', password: 'hashed_password' };
      (sql as any).mockResolvedValue([mockUser]);
      (bcrypt.compare as any).mockResolvedValue(true);
      (jwt.sign as any).mockReturnValue('mock_token');

      const result = await AuthService.login('test@example.com', 'password123');

      expect(result.token).toBe('mock_token');
      expect(result.user.email).toBe('test@example.com');
    });

    it('should throw error if user not found', async () => {
      (sql as any).mockResolvedValue([]);
      await expect(AuthService.login('test@example.com', 'password123')).rejects.toThrow('Invalid email or password');
    });

    it('should throw error if password incorrect', async () => {
      (sql as any).mockResolvedValue([{ id: 1, email: 'test@example.com', password: 'hashed_password' }]);
      (bcrypt.compare as any).mockResolvedValue(false);
      await expect(AuthService.login('test@example.com', 'password123')).rejects.toThrow('Invalid email or password');
    });

    it('should throw error if email or password missing', async () => {
      await expect(AuthService.login('', '')).rejects.toThrow('Email and password are required');
    });
  });
});
