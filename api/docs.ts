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

    // Parse YAML to JSON for Swagger UI
    const spec = YAML.load(specContent) as Record<string, unknown>;

    // HTML for Swagger UI
    const html = `
      <!DOCTYPE html>
      <html lang="en">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>MoneraDigital API Documentation - Swagger UI</title>
        <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.15.5/swagger-ui.min.css">
        <link rel="icon" type="image/png" href="https://fastapi.tiangolo.com/img/favicon.png" sizes="32x32">
        <style>
          html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
          *,
          *:before,
          *:after { box-sizing: inherit; }
          body {
            margin: 0;
            background: #fafafa;
            padding: 20px;
          }
          .swagger-ui {
            max-width: 1460px;
            margin: 0 auto;
          }
          .topbar {
            background-color: #1a1a1a;
            padding: 20px;
            text-align: center;
            color: white;
            border-radius: 4px;
            margin-bottom: 20px;
          }
          .topbar h1 {
            margin: 0;
            font-size: 24px;
          }
          .topbar p {
            margin: 5px 0 0 0;
            color: #ccc;
          }
        </style>
      </head>
      <body>
        <div class="topbar">
          <h1>MoneraDigital Withdrawal API</h1>
          <p>Address Whitelist & Withdrawal Management</p>
        </div>
        <div id="swagger-ui"></div>

        <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.15.5/swagger-ui.min.js"></script>
        <script>
          const spec = ${JSON.stringify(spec)};
          const ui = SwaggerUIBundle({
            spec: spec,
            dom_id: '#swagger-ui',
            presets: [
              SwaggerUIBundle.presets.apis,
              SwaggerUIBundle.SwaggerUIStandalonePreset
            ],
            layout: "StandaloneLayout",
            defaultModelsExpandDepth: 1,
            defaultModelExpandDepth: 1,
          });
        </script>
      </body>
      </html>
    `;

    res.setHeader('Content-Type', 'text/html');
    return res.status(200).send(html);
  } catch (error: any) {
    logger.error({ error: error.message }, 'Failed to serve Swagger UI');
    return res.status(500).json({ error: 'Failed to load API documentation' });
  }
}
