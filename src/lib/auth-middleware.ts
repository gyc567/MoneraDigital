import jwt from 'jsonwebtoken';
import { VercelRequest } from '@vercel/node';

const JWT_SECRET = process.env.JWT_SECRET || 'fallback-secret-for-dev-only';

export interface AuthUser {
  userId: number;
  email: string;
}

export function verifyToken(req: VercelRequest): AuthUser | null {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return null;
  }

  const token = authHeader.split(' ')[1];
  try {
    return jwt.verify(token, JWT_SECRET) as AuthUser;
  } catch (error) {
    return null;
  }
}
