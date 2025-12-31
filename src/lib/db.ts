import postgres from 'postgres';

const connectionString = process.env.DATABASE_URL;

if (!connectionString) {
  console.error('DATABASE_URL is missing! Database connection will fail.');
}

const sql = postgres(connectionString || '', {
  ssl: 'require',
  // Max number of connections
  max: 1,
});

export default sql;
