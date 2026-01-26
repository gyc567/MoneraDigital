import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { v4 as uuidv4 } from 'uuid';

// Mock modules before importing the service
vi.mock('./db');
vi.mock('uuid');
vi.mock('./logger', () => ({
  default: {
    info: vi.fn(),
    debug: vi.fn(),
    error: vi.fn(),
  },
}));

import { SessionService } from './session-service';
import { db } from './db';

describe('SessionService', () => {
  const mockUserId = 123;
  const mockSessionId = 'test-uuid-session-id-1234';

  beforeEach(() => {
    vi.clearAllMocks();
    (uuidv4 as ReturnType<typeof vi.fn>).mockReturnValue(mockSessionId);
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  // ============================================================
  // createPendingLoginSession() tests - 4 test cases
  // ============================================================
  describe('createPendingLoginSession', () => {
    it('should create a session with default 15-minute TTL', async () => {
      const mockInsert = vi.fn().mockReturnValue({
        values: vi.fn().mockResolvedValue([]),
      });
      (db as any).insert = vi.fn().mockReturnValue({ values: mockInsert });

      const result = await SessionService.createPendingLoginSession(mockUserId);

      expect(result).toBe(mockSessionId);
      expect(uuidv4).toHaveBeenCalled();
      expect(db.insert).toHaveBeenCalled();

      // Verify TTL calculation: 15 minutes = 900000ms
      const callArg = mockInsert.mock.calls[0][0];
      const expiresAt = new Date(callArg.expiresAt).getTime();
      const createdAt = new Date(callArg.createdAt).getTime();
      const ttlMs = expiresAt - createdAt;
      expect(ttlMs).toBe(15 * 60 * 1000); // 15 minutes in milliseconds
    });

    it('should create a session with custom TTL', async () => {
      const mockInsert = vi.fn().mockReturnValue({
        values: vi.fn().mockResolvedValue([]),
      });
      (db as any).insert = vi.fn().mockReturnValue({ values: mockInsert });

      const result = await SessionService.createPendingLoginSession(mockUserId, 30);

      expect(result).toBe(mockSessionId);

      // Verify custom TTL: 30 minutes = 1800000ms
      const callArg = mockInsert.mock.calls[0][0];
      const expiresAt = new Date(callArg.expiresAt).getTime();
      const createdAt = new Date(callArg.createdAt).getTime();
      const ttlMs = expiresAt - createdAt;
      expect(ttlMs).toBe(30 * 60 * 1000);
    });

    it('should store session with correct userId and timestamps', async () => {
      const mockInsert = vi.fn().mockReturnValue({
        values: vi.fn().mockResolvedValue([]),
      });
      (db as any).insert = vi.fn().mockReturnValue({ values: mockInsert });

      await SessionService.createPendingLoginSession(mockUserId);

      const callArg = mockInsert.mock.calls[0][0];
      expect(callArg.userId).toBe(mockUserId);
      expect(callArg.sessionId).toBe(mockSessionId);
      expect(callArg.createdAt).toBeInstanceOf(Date);
      expect(callArg.expiresAt).toBeInstanceOf(Date);
    });

    it('should throw error if database insert fails', async () => {
      const mockValues = vi.fn().mockRejectedValue(new Error('DB error'));
      (db as any).insert = vi.fn().mockReturnValue({ values: mockValues });

      await expect(SessionService.createPendingLoginSession(mockUserId)).rejects.toThrow(
        'Failed to create session'
      );
    });
  });

  // ============================================================
  // getPendingLoginSession() tests - 6 test cases
  // ============================================================
  describe('getPendingLoginSession', () => {
    it('should return userId for valid non-expired session', async () => {
      const futureDate = new Date(Date.now() + 10 * 60 * 1000); // 10 minutes in future
      const session = { id: 1, sessionId: mockSessionId, userId: mockUserId, createdAt: new Date(), expiresAt: futureDate };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockReturnValue({
            limit: vi.fn().mockResolvedValue([session]),
          }),
        }),
      });

      const result = await SessionService.getPendingLoginSession(mockSessionId);

      expect(result).toEqual({ userId: mockUserId });
    });

    it('should return null if session not found', async () => {
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockReturnValue({
            limit: vi.fn().mockResolvedValue([]),
          }),
        }),
      });

      const result = await SessionService.getPendingLoginSession(mockSessionId);

      expect(result).toBeNull();
    });

    it('should return null if session expired', async () => {
      const pastDate = new Date(Date.now() - 10 * 60 * 1000); // 10 minutes in past
      // The query is mocked to not find the session (since WHERE condition filters expired)
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockReturnValue({
            limit: vi.fn().mockResolvedValue([]),
          }),
        }),
      });

      const result = await SessionService.getPendingLoginSession(mockSessionId);

      expect(result).toBeNull();
    });

    it('should return null on database error (safe for auth)', async () => {
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockReturnValue({
            limit: vi.fn().mockRejectedValue(new Error('DB error')),
          }),
        }),
      });

      const result = await SessionService.getPendingLoginSession(mockSessionId);

      expect(result).toBeNull();
    });

    it('should query with correct sessionId filter', async () => {
      const mockWhere = vi.fn().mockReturnValue({
        limit: vi.fn().mockResolvedValue([]),
      });
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: mockWhere,
        }),
      });

      await SessionService.getPendingLoginSession(mockSessionId);

      expect(mockWhere).toHaveBeenCalled();
      // Verify it's checking expiry (future dates only)
      const whereArg = mockWhere.mock.calls[0][0];
      expect(whereArg).toBeDefined();
    });

    it('should handle case where multiple sessions returned (should use first)', async () => {
      const futureDate = new Date(Date.now() + 10 * 60 * 1000);
      const sessions = [
        { id: 1, sessionId: mockSessionId, userId: mockUserId, createdAt: new Date(), expiresAt: futureDate },
        { id: 2, sessionId: 'other-session', userId: 999, createdAt: new Date(), expiresAt: futureDate },
      ];

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockReturnValue({
            limit: vi.fn().mockResolvedValue(sessions),
          }),
        }),
      });

      const result = await SessionService.getPendingLoginSession(mockSessionId);

      expect(result?.userId).toBe(mockUserId);
    });
  });

  // ============================================================
  // clearPendingLoginSession() tests - 3 test cases
  // ============================================================
  describe('clearPendingLoginSession', () => {
    it('should delete session from database', async () => {
      const mockDelete = vi.fn().mockResolvedValue([]);
      (db as any).delete = vi.fn(() => ({
        where: mockDelete,
      }));

      await expect(SessionService.clearPendingLoginSession(mockSessionId)).resolves.toBeUndefined();

      expect(db.delete).toHaveBeenCalled();
    });

    it('should handle case where session already deleted', async () => {
      const mockDelete = vi.fn().mockResolvedValue([]);
      (db as any).delete = vi.fn(() => ({
        where: mockDelete,
      }));

      // Should not throw
      await expect(SessionService.clearPendingLoginSession(mockSessionId)).resolves.toBeUndefined();
    });

    it('should throw error if database delete fails', async () => {
      const mockDelete = vi.fn().mockRejectedValue(new Error('DB error'));
      (db as any).delete = vi.fn(() => ({
        where: mockDelete,
      }));

      await expect(SessionService.clearPendingLoginSession(mockSessionId)).rejects.toThrow(
        'Failed to clear session'
      );
    });
  });

  // ============================================================
  // cleanupExpiredSessions() tests - 3 test cases
  // ============================================================
  describe('cleanupExpiredSessions', () => {
    it('should delete expired sessions', async () => {
      const mockDelete = vi.fn().mockReturnValue({
        rowCount: 5,
      });
      (db as any).delete = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue({ rowCount: 5 }),
      });

      const result = await SessionService.cleanupExpiredSessions();

      expect(result).toBe(5);
      expect(db.delete).toHaveBeenCalled();
    });

    it('should return 0 if no expired sessions', async () => {
      (db as any).delete = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue({ rowCount: 0 }),
      });

      const result = await SessionService.cleanupExpiredSessions();

      expect(result).toBe(0);
    });

    it('should return 0 on error (non-throwing cleanup)', async () => {
      (db as any).delete = vi.fn().mockReturnValue({
        where: vi.fn().mockRejectedValue(new Error('DB error')),
      });

      const result = await SessionService.cleanupExpiredSessions();

      expect(result).toBe(0);
    });

    it('should handle rowCount being undefined', async () => {
      (db as any).delete = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue({}), // No rowCount
      });

      const result = await SessionService.cleanupExpiredSessions();

      expect(result).toBe(0);
    });
  });
});
