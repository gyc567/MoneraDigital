import * as bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';
import { z } from 'zod';
import { db } from './db.js';
import { users } from '../db/schema.js';
import { eq } from 'drizzle-orm';
import { decrypt } from './encryption.js';
import logger from './logger.js';

const getJwtSecret = () => {
  const secret = process.env.JWT_SECRET;
  if (!secret) {
    throw new Error('JWT_SECRET environment variable is required');
  }
  return secret;
};

export const authSchema = z.object({
  email: z.string().email('Invalid email format'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
});

export class AuthService {
  /**
   * 注册新用户 (使用 Drizzle ORM)
   */
  static async register(email: string, password: string) {
    const validated = authSchema.parse({ email, password });
    
    logger.info({ email: validated.email }, 'Attempting to register new user');
    
    const hashedPassword = await bcrypt.hash(validated.password, 10);

    try {
      const [user] = await db.insert(users).values({
        email: validated.email,
        password: hashedPassword,
      }).returning({
        id: users.id,
        email: users.email,
      });

      logger.info({ userId: user.id, email: user.email }, 'User registered successfully');
      return user;
    } catch (error: any) {
      // Postgres unique violation code is 23505
      if (error.code === '23505' || error.message?.includes('unique constraint')) {
        logger.warn({ email: validated.email }, 'Registration failed: user already exists');
        throw new Error('User already exists');
      }
      logger.error({ error, email: validated.email }, 'Unexpected error during registration');
      throw error;
    }
  }

  /**
   * 用户登录并生成 JWT (使用 Drizzle ORM)
   */
  static async login(email: string, password: string) {
    const validated = authSchema.parse({ email, password });

    logger.info({ email: validated.email }, 'Login attempt');

    const [user] = await db.select().from(users).where(eq(users.email, validated.email));

    if (!user) {
      logger.warn({ email: validated.email }, 'Login failed: user not found');
      throw new Error('Invalid email or password');
    }

    const isValid = await bcrypt.compare(password, user.password);
    if (!isValid) {
      logger.warn({ email: validated.email, userId: user.id }, 'Login failed: incorrect password');
      throw new Error('Invalid email or password');
    }

    if (user.twoFactorEnabled) {
      logger.info({ userId: user.id }, 'Login requires 2FA');
      return {
        requires2FA: true,
        userId: user.id,
      };
    }

    const token = jwt.sign({ userId: user.id, email: user.email }, getJwtSecret(), {
      expiresIn: '24h',
    });

    logger.info({ userId: user.id, email: user.email }, 'Login successful, token issued');

    return {
      user: { id: user.id, email: user.email },
      token,
    };
  }

  /**
   * 根据用户ID获取用户信息
   */
  static async getUserById(userId: number) {
    const [user] = await db.select({
      id: users.id,
      email: users.email,
    }).from(users).where(eq(users.id, userId));

    return user || null;
  }

  /**
   * 验证 2FA 代码并完成登录
   */
  static async verify2FAAndLogin(userId: number, token: string) {
    const [user] = await db.select().from(users).where(eq(users.id, userId));
    if (!user || !user.twoFactorEnabled) {
      throw new Error('2FA not enabled');
    }

    const { TwoFactorService } = await import('./two-factor-service.js');
    const isValid = await TwoFactorService.verify(userId, token);
    
    if (!isValid) {
      throw new Error('Invalid verification code');
    }

    const jwtToken = jwt.sign({ userId: user.id, email: user.email }, getJwtSecret(), {
      expiresIn: '24h',
    });

    return {
      user: { id: user.id, email: user.email },
      token: jwtToken,
    };
  }
}