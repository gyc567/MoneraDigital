import type { VercelRequest, VercelResponse } from '@vercel/node';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // Go 后端地址
  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  try {
    // 纯转发到 Go 后端
    const response = await fetch(`${backendUrl}/api/auth/2fa/status`, {
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
    console.error('Proxy error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
