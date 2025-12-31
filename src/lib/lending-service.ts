import { db } from './db.js';
import { lendingPositions } from '../db/schema.js';
import { eq, and } from 'drizzle-orm';
import logger from './logger.js';

export class LendingService {
  /**
   * 根据币种和期限计算 APY
   * KISS 原则：暂时使用静态映射，后续可对接市场接口
   */
  static calculateAPY(asset: string, durationDays: number): number {
    const baseRates: Record<string, number> = {
      'BTC': 4.5,
      'ETH': 5.2,
      'USDT': 8.5,
      'USDC': 8.2,
      'SOL': 6.8,
    };

    const multiplier = durationDays >= 360 ? 1.5 : durationDays >= 180 ? 1.25 : durationDays >= 90 ? 1.1 : 1.0;
    return (baseRates[asset] || 5.0) * multiplier;
  }

  /**
   * 计算预估总收益
   */
  static calculateEstimatedYield(amount: number, apy: number, durationDays: number): number {
    return (amount * (apy / 100) * durationDays) / 365;
  }

  /**
   * 创建借贷头寸
   */
  static async applyForLending(userId: number, asset: string, amount: number, durationDays: number) {
    const apy = this.calculateAPY(asset, durationDays);
    const startDate = new Date();
    const endDate = new Date();
    endDate.setDate(startDate.getDate() + durationDays);

    logger.info({ userId, asset, amount, durationDays, apy }, 'Processing lending application');

    try {
      const [position] = await db.insert(lendingPositions).values({
        userId,
        asset,
        amount: amount.toString(),
        durationDays,
        apy: apy.toFixed(2),
        status: 'ACTIVE',
        endDate,
      }).returning();

      return position;
    } catch (error) {
      logger.error({ error, userId }, 'Failed to create lending position');
      throw new Error('Lending application failed');
    }
  }

  /**
   * 获取用户活跃头寸
   */
  static async getUserPositions(userId: number) {
    return db.select()
      .from(lendingPositions)
      .where(eq(lendingPositions.userId, userId))
      .orderBy(lendingPositions.startDate);
  }

  /**
   * 提前终止
   */
  static async terminatePosition(userId: number, positionId: number) {
    logger.info({ userId, positionId }, 'Attempting early termination');
    
    const [position] = await db.select()
      .from(lendingPositions)
      .where(and(eq(lendingPositions.id, positionId), eq(lendingPositions.userId, userId)));

    if (!position) throw new Error('Position not found');
    if (position.status !== 'ACTIVE') throw new Error('Position is not active');

    return db.update(lendingPositions)
      .set({ status: 'TERMINATED' })
      .where(eq(lendingPositions.id, positionId))
      .returning();
  }
}
