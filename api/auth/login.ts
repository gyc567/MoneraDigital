import type { VercelRequest, VercelResponse } from '@vercel/node';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // Go后端地址 - Use BACKEND_URL for Vercel functions, fallback to VITE_API_BASE_URL for dev
  const backendUrl = process.env.BACKEND_URL || process.env.VITE_API_BASE_URL || 'http://localhost:8081';

  // Allow localhost in development, but require proper URL in production
  const isProduction = process.env.NODE_ENV === 'production' || process.env.VERCEL_ENV === 'production';
  if (isProduction && (!backendUrl || backendUrl === 'http://localhost:8081')) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  try {
    // 纯转发到Go后端
    const response = await fetch(`${backendUrl}/api/auth/login`, {
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
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
