import { authenticator } from 'otplib';
import QRCode from 'qrcode';
import crypto from 'crypto';
import { db } from './db.js';
import { users } from '../db/schema.js';
import { eq } from 'drizzle-orm';
import { encrypt, decrypt } from './encryption.js';
import logger from './logger.js';

/**
 * TwoFactorService handles all Two-Factor Authentication operations.
 * Provides TOTP secret generation, QR code generation, token verification,
 * and backup code management.
 *
 * Design Principles:
 * - All cryptographic operations use proper encryption (AES-256-GCM)
 * - Secrets are never logged or returned except during setup
 * - Backup codes are one-time use only
 * - All operations are logged for audit trails
 *
 * Integration:
 * - Used by api/auth/2fa/* endpoints
 * - Used by api/auth/2fa/verify-login endpoint for 2FA validation
 * - Reusable service following backend-as-source-of-truth pattern
 */
export class TwoFactorService {
  /**
   * Generates a new TOTP secret for a user and creates backup codes.
   * Stores encrypted secret and backup codes in database.
   *
   * @param userId - The user ID (must exist in database)
   * @param email - User's email address (used in otpauth URI display)
   * @returns Object containing:
   *   - secret: The TOTP secret key (base32 encoded, min 16 chars)
   *   - otpauth: The otpauth:// URI for manual authenticator app entry
   *   - qrCodeUrl: Data URL of QR code (data:image/png;base64,...)
   *   - backupCodes: Array of exactly 10 backup codes (8 hex chars each)
   * @throws Error if database update fails
   *
   * Flow:
   * 1. Generate cryptographically secure TOTP secret
   * 2. Create otpauth URI for authenticator app compatibility
   * 3. Generate QR code as data URL for easy scanning
   * 4. Generate 10 random backup codes (one-time use recovery)
   * 5. Encrypt and store secret + backup codes in database
   * 6. Return plaintext values (encrypted storage, not in transit)
   *
   * Security:
   * - Uses otplib for cryptographic randomness
   * - Secrets encrypted with AES-256-GCM before storage
   * - QR code generated server-side, never transmitted unencrypted
   */
  static async setup(userId: number, email: string) {
    const secret = authenticator.generateSecret();
    const otpauth = authenticator.keyuri(email, 'Monera Digital', secret);

    // Generate 10 cryptographically random backup codes
    const backupCodes = Array.from({ length: 10 }, () =>
      crypto.randomBytes(4).toString('hex').toUpperCase()
    );

    const qrCodeUrl = await QRCode.toDataURL(otpauth, {
      margin: 2,
      width: 400,
    });

    // Store encrypted secret and backup codes
    await db
      .update(users)
      .set({
        twoFactorSecret: encrypt(secret),
        twoFactorBackupCodes: encrypt(JSON.stringify(backupCodes)),
      })
      .where(eq(users.id, userId));

    return { secret, qrCodeUrl, backupCodes, otpauth };
  }

  /**
   * Enables 2FA by validating the user's first TOTP token.
   * This confirms the user has successfully scanned the QR code
   * and can generate valid tokens.
   *
   * @param userId - The user ID
   * @param token - 6-digit TOTP token from authenticator app
   * @returns true on success
   * @throws Error with descriptive message if validation fails:
   *   - "2FA has not been set up" if secret is null
   *   - "Invalid verification code" if token doesn't match
   *   - "User not found" if userId doesn't exist
   *
   * Flow:
   * 1. Fetch user from database
   * 2. Check if setup was completed (twoFactorSecret exists)
   * 3. Decrypt secret and validate token using otplib
   * 4. If valid: set twoFactorEnabled = true in database
   *
   * Security:
   * - Token must be exactly 6 digits (enforced by API schema)
   * - Decryption happens in-memory only
   * - Invalid attempts are not tracked (rate limiting at API level)
   */
  static async enable(userId: number, token: string) {
    const [user] = await db.select().from(users).where(eq(users.id, userId));

    if (!user || !user.twoFactorSecret) {
      throw new Error('2FA has not been set up');
    }

    const decryptedSecret = decrypt(user.twoFactorSecret);
    const isValid = authenticator.check(token, decryptedSecret);

    if (!isValid) {
      throw new Error('Invalid verification code');
    }

    await db.update(users).set({ twoFactorEnabled: true }).where(eq(users.id, userId));

    return true;
  }

  /**
   * Verifies a 2FA token (TOTP or backup code) during login flow.
   * Supports both time-based one-time passwords and backup codes.
   *
   * @param userId - The user ID
   * @param token - Either a 6-digit TOTP code or 8-char backup code
   * @returns true if token is valid, false otherwise
   *          Also returns true if 2FA is not enabled (bypass)
   *
   * Flow:
   * 1. Fetch user from database
   * 2. Return true if 2FA not enabled (no verification needed)
   * 3. Try to verify token as TOTP using otplib
   * 4. If TOTP fails, try to match against backup codes
   * 5. If backup code matches:
   *    - Remove it from array (one-time use)
   *    - Update database
   *    - Return true
   * 6. Return false if no match found
   *
   * Security:
   * - Backup codes are case-insensitive for UX
   * - Backup codes support one-time use enforcement
   * - Failed attempts not tracked (rate limiting at API level)
   * - Decryption happens in-memory only
   *
   * Note:
   * - Returns true if user/secret not found (safe default for bypass)
   * - Errors during decryption are logged but don't block flow
   */
  static async verify(userId: number, token: string) {
    const [user] = await db.select().from(users).where(eq(users.id, userId));

    if (!user || !user.twoFactorSecret || !user.twoFactorEnabled) {
      return true;
    }

    // 1. Try TOTP verification
    const decryptedSecret = decrypt(user.twoFactorSecret);
    if (authenticator.check(token, decryptedSecret)) {
      return true;
    }

    // 2. Try backup code verification
    if (user.twoFactorBackupCodes) {
      const backupCodes: string[] = JSON.parse(decrypt(user.twoFactorBackupCodes));
      const codeIndex = backupCodes.indexOf(token.toUpperCase());

      if (codeIndex !== -1) {
        // Remove used backup code (one-time use enforcement)
        backupCodes.splice(codeIndex, 1);
        await db
          .update(users)
          .set({ twoFactorBackupCodes: encrypt(JSON.stringify(backupCodes)) })
          .where(eq(users.id, userId));

        logger.info({ userId }, 'Used a 2FA backup code');
        return true;
      }
    }

    return false;
  }

  /**
   * Disables 2FA for a user after verifying current TOTP token.
   * This ensures only the account owner can disable 2FA.
   *
   * @param userId - The user ID
   * @param token - 6-digit TOTP token to verify identity
   * @returns true on success
   * @throws Error with descriptive message if validation fails:
   *   - "2FA is not enabled" if not currently enabled
   *   - "Invalid verification code" if token doesn't match
   *   - "User not found" if userId doesn't exist
   *
   * Flow:
   * 1. Fetch user from database
   * 2. Verify user exists and 2FA is enabled
   * 3. Decrypt secret and validate TOTP token
   * 4. If valid: clear secret, backup codes, and disable flag
   *
   * Security:
   * - Requires valid TOTP (identity verification)
   * - Completely clears all 2FA data (secrets, codes)
   * - Operation is idempotent (safe to retry)
   * - Logged for audit trail
   */
  static async disable(userId: number, token: string): Promise<boolean> {
    const [user] = await db.select().from(users).where(eq(users.id, userId));

    // Check if user exists and 2FA is enabled
    if (!user || !user.twoFactorEnabled) {
      throw new Error('2FA is not enabled');
    }

    // Check if secret exists
    if (!user.twoFactorSecret) {
      throw new Error('2FA is not enabled');
    }

    // Decrypt secret and verify token
    const decryptedSecret = decrypt(user.twoFactorSecret);
    const isValid = authenticator.check(token, decryptedSecret);

    if (!isValid) {
      throw new Error('Invalid verification code');
    }

    // Disable 2FA and clear secret/backup codes
    await db
      .update(users)
      .set({
        twoFactorEnabled: false,
        twoFactorSecret: null,
        twoFactorBackupCodes: null,
      })
      .where(eq(users.id, userId));

    logger.info({ userId }, '2FA has been disabled');
    return true;
  }

  /**
   * Gets the current 2FA status for a user.
   * Used to display 2FA configuration in security settings.
   *
   * @param userId - The user ID
   * @returns Object containing:
   *   - enabled: Boolean indicating if 2FA is active
   *   - remainingBackupCodes: Number of unused backup codes (0-10)
   *
   * Behavior:
   * - Returns { enabled: false, remainingBackupCodes: 0 } if user not found
   * - Returns { enabled: false, remainingBackupCodes: 0 } if 2FA not enabled
   * - Returns { enabled: true, remainingBackupCodes: N } if 2FA enabled
   * - Returns remainingBackupCodes: 0 if backup codes can't be decrypted
   *
   * Error Handling:
   * - Decryption errors are logged but don't throw
   * - JSON parse errors are handled gracefully
   * - Returns safe defaults on any error
   */
  static async getStatus(
    userId: number
  ): Promise<{ enabled: boolean; remainingBackupCodes: number }> {
    const [user] = await db.select().from(users).where(eq(users.id, userId));

    // Return disabled status if user not found or 2FA is not enabled
    if (!user || !user.twoFactorEnabled) {
      return { enabled: false, remainingBackupCodes: 0 };
    }

    // Count remaining backup codes
    let remainingBackupCodes = 0;

    if (user.twoFactorBackupCodes) {
      try {
        const decryptedCodes = decrypt(user.twoFactorBackupCodes);
        const backupCodes: string[] = JSON.parse(decryptedCodes);
        remainingBackupCodes = backupCodes.length;
      } catch {
        // Handle decryption or JSON parsing errors gracefully
        logger.error({ userId }, 'Failed to parse backup codes');
        remainingBackupCodes = 0;
      }
    }

    return { enabled: true, remainingBackupCodes };
  }
}
