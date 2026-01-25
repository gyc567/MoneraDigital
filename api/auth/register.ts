import type { VercelRequest, VercelResponse } from '@vercel/node';

// Go后端地址
const GO_BACKEND_URL = process.env.GO_BACKEND_URL || 'http://localhost:8081';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    // 纯转发到Go后端
    const response = await fetch(`${GO_BACKEND_URL}/api/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(req.body),
    });

    const data = await response.json();

    // 转发响应状态和内容
    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Proxy error:', errorMessage);
    return res.status(500).json({ error: 'Internal Server Error' });
  }
}
