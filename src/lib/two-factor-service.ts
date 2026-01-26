import { authenticator } from 'otplib';
import QRCode from 'qrcode';
import crypto from 'crypto';
import { db } from './db.js';
import { users } from '../db/schema.js';
import { eq } from 'drizzle-orm';
import { encrypt, decrypt } from './encryption.js';
import logger from './logger.js';

export class TwoFactorService {
  /**
   * 生成新的 2FA 密钥、二维码和恢复码
   */
  static async setup(userId: number, email: string) {
    const secret = authenticator.generateSecret();
    const otpauth = authenticator.keyuri(email, 'Monera Digital', secret);
    
    // 生成 10 个恢复码
    const backupCodes = Array.from({ length: 10 }, () => 
      crypto.randomBytes(4).toString('hex').toUpperCase()
    );
    
    const qrCodeUrl = await QRCode.toDataURL(otpauth, {
      margin: 2,
      width: 400
    });
    
    // 存储加密后的密钥和恢复码
    await db.update(users)
      .set({ 
        twoFactorSecret: encrypt(secret),
        twoFactorBackupCodes: encrypt(JSON.stringify(backupCodes))
      })
      .where(eq(users.id, userId));

    return { secret, qrCodeUrl, backupCodes, otpauth };
  }

  /**
   * 验证并正式启用 2FA
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

    await db.update(users)
      .set({ twoFactorEnabled: true })
      .where(eq(users.id, userId));
    
    return true;
  }

  /**
   * 验证代码 (支持主验证码和恢复码)
   */
  static async verify(userId: number, token: string) {
    const [user] = await db.select().from(users).where(eq(users.id, userId));

    if (!user || !user.twoFactorSecret || !user.twoFactorEnabled) {
      return true;
    }

    // 1. 尝试验证 TOTP
    const decryptedSecret = decrypt(user.twoFactorSecret);
    if (authenticator.check(token, decryptedSecret)) {
      return true;
    }

    // 2. 尝试验证恢复码
    if (user.twoFactorBackupCodes) {
      const backupCodes: string[] = JSON.parse(decrypt(user.twoFactorBackupCodes));
      const codeIndex = backupCodes.indexOf(token.toUpperCase());

      if (codeIndex !== -1) {
        // 恢复码正确，将其移除并更新数据库
        backupCodes.splice(codeIndex, 1);
        await db.update(users)
          .set({ twoFactorBackupCodes: encrypt(JSON.stringify(backupCodes)) })
          .where(eq(users.id, userId));

        logger.info({ userId }, 'Used a 2FA backup code');
        return true;
      }
    }

    return false;
  }

  /**
   * 禁用 2FA (需要验证当前 TOTP token)
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
    await db.update(users)
      .set({
        twoFactorEnabled: false,
        twoFactorSecret: null,
        twoFactorBackupCodes: null
      })
      .where(eq(users.id, userId));

    logger.info({ userId }, '2FA has been disabled');
    return true;
  }

  /**
   * 获取 2FA 状态
   */
  static async getStatus(userId: number): Promise<{ enabled: boolean; remainingBackupCodes: number }> {
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