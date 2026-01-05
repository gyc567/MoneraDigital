import type { VercelRequest, VercelResponse } from '@vercel/node';
import fs from 'fs';
import path from 'path';
import YAML from 'js-yaml';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    // Read the OpenAPI spec file
    const specPath = path.join(process.cwd(), 'docs', 'openapi.yaml');
    const specContent = fs.readFileSync(specPath, 'utf-8');

    // Parse YAML to JSON
    const spec = YAML.load(specContent);

    res.setHeader('Content-Type', 'application/json');
    return res.status(200).json(spec);
  } catch (error: any) {
    logger.error({ error: error.message }, 'Failed to serve OpenAPI spec');
    return res.status(500).json({ error: 'Failed to load API specification' });
  }
}
