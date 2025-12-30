import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';
import { z } from 'zod';
import sql from './db.js';

const JWT_SECRET = process.env.JWT_SECRET;
if (!JWT_SECRET) {
  throw new Error('JWT_SECRET environment variable is required');
}

export const authSchema = z.object({
  email: z.string().email('Invalid email format'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
});

export class AuthService {
  /**
   * 注册新用户
   */
  static async register(email: string, password: string) {
    const validated = authSchema.parse({ email, password });
    
    const hashedPassword = await bcrypt.hash(validated.password, 10);

    try {
      const [user] = await sql`
        INSERT INTO users (email, password)
        VALUES (${email}, ${hashedPassword})
        RETURNING id, email
      `;
      return user;
    } catch (error: any) {
      if (error.code === '23505') {
        throw new Error('User already exists');
      }
      throw error;
    }
  }

  /**
   * 用户登录并生成 JWT
   */
  static async login(email: string, password: string) {
    const validated = authSchema.parse({ email, password });

    const [user] = await sql`
      SELECT id, email, password FROM users WHERE email = ${validated.email}
    `;

    if (!user) {
      throw new Error('Invalid email or password');
    }

    const isValid = await bcrypt.compare(password, user.password);
    if (!isValid) {
      throw new Error('Invalid email or password');
    }

    const token = jwt.sign({ userId: user.id, email: user.email }, JWT_SECRET, {
      expiresIn: '24h',
    });

    return {
      user: { id: user.id, email: user.email },
      token,
    };
  }
}
