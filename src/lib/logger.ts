import pino from 'pino';

const logger = pino({
  level: 'info',
  base: {
    env: 'production',
    service: 'monera-digital-frontend',
  },
});

export default logger;
