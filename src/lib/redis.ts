import { Redis } from '@upstash/redis';
import logger from './logger.js';

const redisUrl = import.meta.env.VITE_UPSTASH_REDIS_REST_URL;
const redisToken = import.meta.env.VITE_UPSTASH_REDIS_REST_TOKEN;

if (!redisUrl || !redisToken) {
  logger.warn('Redis credentials missing (VITE_UPSTASH_REDIS_*). Distributed rate limiting will be disabled.');
}

export const redis = (redisUrl && redisToken)
  ? new Redis({
      url: redisUrl,
      token: redisToken,
    })
  : null;
