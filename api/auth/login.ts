import type { VercelRequest, VercelResponse } from '@vercel/node';
import { LoginResponseSchema } from '../../src/lib/two-factor-schemas.js';
import logger from '../../src/lib/logger.js';

/**
 * POST /api/auth/login
 *
 * Pure HTTP proxy to Go backend for user authentication.
 * The Go backend handles password validation and determines if 2FA is required.
 *
 * Go Response Types:
 * 1. Successful login without 2FA: { success: true, token, user: { id, email } }
 * 2. Requires 2FA: { requires2FA: true, sessionId, user: { id, email } }
 */
export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // Go后端地址
  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.',
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

    // Validate response structure (optional but helps catch unexpected Go backend changes)
    const validated = LoginResponseSchema.safeParse(data);

    if (!validated.success) {
      logger.warn(
        { error: validated.error.issues, goResponse: data },
        'Go backend login response does not match expected schema'
      );
      // Still return the Go response even if validation fails
      // This ensures compatibility while logging the issue
    }

    // 转发响应状态和内容
    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Login proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
