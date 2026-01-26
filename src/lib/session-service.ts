import { v4 as uuidv4 } from 'uuid';
import { db } from './db.js';
import { pendingLoginSessions, users } from '../db/schema.js';
import { eq, and, gt } from 'drizzle-orm';
import logger from './logger.js';

/**
 * SessionService manages temporary login sessions for 2FA verification.
 * Sessions are used to track incomplete login attempts waiting for 2FA confirmation.
 *
 * Lifecycle:
 * 1. After password validation, if 2FA enabled: create pending session
 * 2. Return sessionId to client
 * 3. Client sends sessionId + TOTP/backup code to verify endpoint
 * 4. On successful verification: clear session and issue JWT
 * 5. On expiry: session automatically invalid after TTL
 */
export class SessionService {
  /**
   * Creates a new pending login session for a user with 2FA enabled.
   * Session will automatically expire after 15 minutes.
   *
   * @param userId - The user ID from authentication
   * @param ttlMinutes - Time-to-live in minutes (default: 15)
   * @returns Unique session ID for tracking this 2FA attempt
   * @throws Error if database insert fails
   */
  static async createPendingLoginSession(userId: number, ttlMinutes: number = 15): Promise<string> {
    try {
      const sessionId = uuidv4();
      const now = new Date();
      const expiresAt = new Date(now.getTime() + ttlMinutes * 60000);

      await db.insert(pendingLoginSessions).values({
        sessionId,
        userId,
        createdAt: now,
        expiresAt,
      });

      logger.info(
        { userId, sessionId, expiresAt },
        `Created pending login session (TTL: ${ttlMinutes}m)`
      );

      return sessionId;
    } catch (error) {
      logger.error({ userId, error }, 'Failed to create pending login session');
      throw new Error('Failed to create session');
    }
  }

  /**
   * Retrieves an active pending login session by sessionId.
   * Returns null if session not found or has expired.
   *
   * @param sessionId - The session ID to lookup
   * @returns Object with userId if session is valid, null if expired/missing
   * @throws No throws - returns null on failure (safe for auth flows)
   */
  static async getPendingLoginSession(sessionId: string): Promise<{ userId: number } | null> {
    try {
      const now = new Date();

      const [session] = await db
        .select()
        .from(pendingLoginSessions)
        .where(
          and(
            eq(pendingLoginSessions.sessionId, sessionId),
            gt(pendingLoginSessions.expiresAt, now) // Check expiry: expiresAt > now
          )
        )
        .limit(1);

      if (!session) {
        logger.debug({ sessionId }, 'Pending login session not found or expired');
        return null;
      }

      logger.debug({ sessionId, userId: session.userId }, 'Retrieved valid pending login session');
      return { userId: session.userId };
    } catch (error) {
      logger.error({ sessionId, error }, 'Failed to retrieve pending login session');
      return null;
    }
  }

  /**
   * Clears an active pending login session after successful 2FA verification.
   * This prevents session reuse after JWT is issued.
   *
   * @param sessionId - The session ID to clear
   * @throws Error if database delete fails
   */
  static async clearPendingLoginSession(sessionId: string): Promise<void> {
    try {
      await db.delete(pendingLoginSessions).where(eq(pendingLoginSessions.sessionId, sessionId));

      logger.info({ sessionId }, 'Cleared pending login session');
    } catch (error) {
      logger.error({ sessionId, error }, 'Failed to clear pending login session');
      throw new Error('Failed to clear session');
    }
  }

  /**
   * Cleanup expired sessions (housekeeping).
   * Called periodically to remove sessions older than their TTL.
   * Prevents database bloat over time.
   *
   * @returns Number of expired sessions deleted
   */
  static async cleanupExpiredSessions(): Promise<number> {
    try {
      const now = new Date();

      const result = await db
        .delete(pendingLoginSessions)
        .where(eq(pendingLoginSessions.expiresAt < now));

      const deletedCount = result.rowCount;
      logger.info({ deletedCount }, 'Cleaned up expired pending login sessions');

      return deletedCount || 0;
    } catch (error) {
      logger.error({ error }, 'Failed to cleanup expired sessions');
      return 0;
    }
  }
}
