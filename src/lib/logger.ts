import pino from 'pino';

const LOG_LEVEL = import.meta.env.VITE_LOG_LEVEL || 'info';

const logger = pino({
  level: LOG_LEVEL,
  base: {
    env: import.meta.env.DEV ? 'development' : 'production',
    service: 'monera-digital-frontend',
  },
});

export default logger;
