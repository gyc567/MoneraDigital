import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import * as schema from '../db/schema.js';
import logger from './logger.js';

const connectionString = process.env.DATABASE_URL;

if (!connectionString) {
  logger.error('DATABASE_URL is missing!');
}

// 原始连接，用于 Drizzle
export const client = postgres(connectionString || '', { 
  ssl: 'require',
  max: 1 
});

// Drizzle 实例
export const db = drizzle(client, { schema });

// 默认导出旧版 sql 以保持兼容性
export default client;