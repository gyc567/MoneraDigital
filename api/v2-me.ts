import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../src/lib/auth-middleware.js';
import { db } from '../src/lib/db.js';
import { users } from '../src/db/schema.js';
import { eq } from 'drizzle-orm';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const [dbUser] = await db.select({
      id: users.id,
      email: users.email,
      twoFactorEnabled: users.twoFactorEnabled,
    }).from(users).where(eq(users.id, user.userId));

    if (!dbUser) {
      return res.status(404).json({ error: 'User not found' });
    }

    res.status(200).json(dbUser);
  } catch (error: any) {
    res.status(500).json({ error: 'Internal Server Error' });
  }
}
