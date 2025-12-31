import { authenticator } from 'otplib';
import QRCode from 'qrcode';
import { db } from './db.js';
import { users } from '../db/schema.js';
import { eq } from 'drizzle-orm';
import logger from './logger.js';

export class TwoFactorService {
  /**
   * 生成新的 2FA 密钥和二维码
   */
  static async setup(userId: number, email: string) {
    const secret = authenticator.generateSecret();
    const otpauth = authenticator.keyuri(email, 'Monera Digital', secret);
    
    const qrCodeUrl = await QRCode.toDataURL(otpauth);
    
    logger.info({ userId }, 'Generated new 2FA secret');
    
    // 暂时存入数据库（此时尚未启用）
    await db.update(users)
      .set({ twoFactorSecret: secret })
      .where(eq(users.id, userId));

    return { secret, qrCodeUrl };
  }

  /**
   * 验证并正式启用 2FA
   */
  static async enable(userId: number, token: string) {
    const [user] = await db.select().from(users).where(eq(users.id, userId));
    
    if (!user || !user.twoFactorSecret) {
      throw new Error('2FA has not been set up');
    }

    const isValid = authenticator.check(token, user.twoFactorSecret);
    if (!isValid) {
      logger.warn({ userId }, '2FA enable failed: invalid token');
      throw new Error('Invalid verification code');
    }

    await db.update(users)
      .set({ twoFactorEnabled: true })
      .where(eq(users.id, userId));
    
    logger.info({ userId }, '2FA enabled successfully');
    return true;
  }

  /**
   * 验证登录时的 2FA 代码
   */
  static async verify(userId: number, token: string) {
    const [user] = await db.select().from(users).where(eq(users.id, userId));
    
    if (!user || !user.twoFactorSecret || !user.twoFactorEnabled) {
      // 如果没开启 2FA，默认通过（理论上业务层应先判断）
      return true;
    }

    return authenticator.check(token, user.twoFactorSecret);
  }

  /**
   * 关闭 2FA
   */
  static async disable(userId: number, token: string) {
    const isValid = await this.verify(userId, token);
    if (!isValid) {
      throw new Error('Invalid verification code');
    }

    await db.update(users)
      .set({ twoFactorEnabled: false, twoFactorSecret: null })
      .where(eq(users.id, userId));
    
    logger.info({ userId }, '2FA disabled');
    return true;
  }
}
