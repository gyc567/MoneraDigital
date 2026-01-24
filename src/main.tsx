import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import "./index.css";
import './i18n';
import { validateDevEnvironment, logDevConfig } from './lib/dev-config-validator';

// Validate development environment on startup
if (import.meta.env.DEV) {
  validateDevEnvironment().catch(err => {
    console.error('[Dev Config] Validation error:', err);
  });
  logDevConfig();
}

createRoot(document.getElementById("root")!).render(<App />);
