import type { VercelRequest, VercelResponse } from '@vercel/node';
import logger from '../../src/lib/logger.js';

/**
 * /api/addresses/[...path]
 *
 * Proxy to Go backend for address sub-resources.
 * Handles:
 * - DELETE /api/addresses/:id
 * - POST /api/addresses/:id/verify
 * - POST /api/addresses/:id/primary
 */
export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;
  const { path } = req.query;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  try {
    // Construct target URL
    // path is an array of path segments e.g. ["1", "verify"]
    const pathStr = Array.isArray(path) ? path.join('/') : path;
    const url = `${backendUrl}/api/addresses/${pathStr}`;
    
    const options: RequestInit = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        ...(req.headers.authorization ? { 'Authorization': req.headers.authorization } : {}),
      },
    };

    if (req.body && (req.method === 'POST' || req.method === 'PUT' || req.method === 'PATCH')) {
      options.body = JSON.stringify(req.body);
    }

    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Addresses sub-path proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
