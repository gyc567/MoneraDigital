import postgres from 'postgres';

const connectionString = process.env.DATABASE_URL || '';
const sql = postgres(connectionString, {
  ssl: 'require',
  // Max number of connections
  max: 1,
});

export default sql;
