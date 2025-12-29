import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';
import sql from './db';

const JWT_SECRET = process.env.JWT_SECRET || 'fallback-secret';

export class AuthService {
  /**
   * 注册新用户
   */
  static async register(email: string, password: string) {
    if (!email || !password) {
      throw new Error('Email and password are required');
    }

    const hashedPassword = await bcrypt.hash(password, 10);

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
    if (!email || !password) {
      throw new Error('Email and password are required');
    }

    const [user] = await sql`
      SELECT id, email, password FROM users WHERE email = ${email}
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
