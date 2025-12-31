import { Redis } from '@upstash/redis';
import logger from './logger.js';

const redisUrl = process.env.UPSTASH_REDIS_REST_URL;
const redisToken = process.env.UPSTASH_REDIS_REST_TOKEN;

if (!redisUrl || !redisToken) {
  logger.warn('Redis credentials missing. Distributed rate limiting will be disabled.');
}

export const redis = (redisUrl && redisToken) 
  ? new Redis({
      url: redisUrl,
      token: redisToken,
    })
  : null;
