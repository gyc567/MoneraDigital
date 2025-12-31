import { redis } from './redis.js';
import logger from './logger.js';

const memoryCache = new Map<string, { count: number; expires: number }>();

/**
 * 分布式频率限制
 */
export async function rateLimit(ip: string, limit: number, windowMs: number): Promise<boolean> {
  // 如果配置了 Redis，使用分布式限流
  if (redis) {
    try {
      const key = `ratelimit:${ip}`;
      const count = await redis.incr(key);
      
      if (count === 1) {
        await redis.pexpire(key, windowMs);
      }
      
      return count <= limit;
    } catch (error) {
      logger.error('Redis rate limiting error, falling back to memory:', error);
    }
  }

  // 备用方案：本地内存限流
  const now = Date.now();
  const record = memoryCache.get(ip);

  if (!record || now > record.expires) {
    memoryCache.set(ip, { count: 1, expires: now + windowMs });
    return true;
  }

  if (record.count >= limit) {
    return false;
  }

  record.count++;
  return true;
}