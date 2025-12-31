import { pgTable, serial, text, timestamp, numeric, integer, pgEnum, boolean } from 'drizzle-orm/pg-core';

export const lendingStatusEnum = pgEnum('lending_status', ['ACTIVE', 'COMPLETED', 'TERMINATED']);

export const users = pgTable('users', {
  id: serial('id').primaryKey(),
  email: text('email').notNull().unique(),
  password: text('password').notNull(),
  twoFactorSecret: text('two_factor_secret'),
  twoFactorEnabled: boolean('two_factor_enabled').default(false).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
});

export const lendingPositions = pgTable('lending_positions', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  asset: text('asset').notNull(),
  amount: numeric('amount', { precision: 20, scale: 8 }).notNull(),
  durationDays: integer('duration_days').notNull(),
  apy: numeric('apy', { precision: 5, scale: 2 }).notNull(),
  status: lendingStatusEnum('status').default('ACTIVE').notNull(),
  accruedYield: numeric('accrued_yield', { precision: 20, scale: 8 }).default('0').notNull(),
  startDate: timestamp('start_date').defaultNow().notNull(),
  endDate: timestamp('end_date').notNull(),
});

export type User = typeof users.$inferSelect;
export type LendingPosition = typeof lendingPositions.$inferSelect;
export type NewLendingPosition = typeof lendingPositions.$inferInsert;
