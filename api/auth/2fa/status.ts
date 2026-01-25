import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../../src/lib/auth-middleware.js';

// Go后端地址 - 统一使用VITE_API_BASE_URL
const BACKEND_URL = process.env.VITE_API_BASE_URL || 'http://localhost:8081';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // 验证JWT令牌
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'AUTH_REQUIRED', message: 'Authentication required' });
  }

  try {
    // 纯转发到Go后端
    const response = await fetch(`${BACKEND_URL}/api/auth/2fa/status`, {
      method: 'GET',
      headers: {
        'Authorization': req.headers.authorization || '',
        'Content-Type': 'application/json',
      },
    });

    const data = await response.json();

    // 转发响应状态和内容
    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('2FA Status proxy error:', errorMessage);
    return res.status(500).json({ error: 'Internal Server Error' });
  }
}
