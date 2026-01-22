import express from 'express';
import { createServer as createViteServer } from 'vite';

const app = express();
app.use(express.json());

// Create Vite dev server
const vite = await createViteServer({
  server: { middlewareMode: true }
});

app.use(vite.middlewares);

app.listen(8081, () => {
  console.log('Dev server running on http://localhost:8081');
});
