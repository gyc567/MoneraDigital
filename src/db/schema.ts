import { pgTable, serial, text, timestamp, numeric, integer, pgEnum, boolean } from 'drizzle-orm/pg-core';

export const lendingStatusEnum = pgEnum('lending_status', ['ACTIVE', 'COMPLETED', 'TERMINATED']);
export const addressTypeEnum = pgEnum('address_type', ['BTC', 'ETH', 'USDC', 'USDT']);
export const withdrawalStatusEnum = pgEnum('withdrawal_status', ['PENDING', 'PROCESSING', 'COMPLETED', 'FAILED']);
export const walletCreationStatusEnum = pgEnum('wallet_creation_status', ['CREATING', 'SUCCESS', 'FAILED']);

export const users = pgTable('users', {
  id: serial('id').primaryKey(),
  email: text('email').notNull().unique(),
  password: text('password').notNull(),
  twoFactorSecret: text('two_factor_secret'),
  twoFactorEnabled: boolean('two_factor_enabled').default(false).notNull(),
  twoFactorBackupCodes: text('two_factor_backup_codes'),
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

export const withdrawalAddresses = pgTable('withdrawal_addresses', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  address: text('address').notNull(),
  addressType: addressTypeEnum('address_type').notNull(),
  label: text('label').notNull(),
  isVerified: boolean('is_verified').default(false).notNull(),
  isPrimary: boolean('is_primary').default(false).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  verifiedAt: timestamp('verified_at'),
  deactivatedAt: timestamp('deactivated_at'),
});

export const addressVerifications = pgTable('address_verifications', {
  id: serial('id').primaryKey(),
  addressId: integer('address_id').references(() => withdrawalAddresses.id).notNull(),
  token: text('token').notNull().unique(),
  expiresAt: timestamp('expires_at').notNull(),
  verifiedAt: timestamp('verified_at'),
});

export const withdrawals = pgTable('withdrawals', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  fromAddressId: integer('from_address_id').references(() => withdrawalAddresses.id).notNull(),
  amount: numeric('amount', { precision: 20, scale: 8 }).notNull(),
  asset: text('asset').notNull(),
  toAddress: text('to_address').notNull(),
  status: withdrawalStatusEnum('status').default('PENDING').notNull(),
  txHash: text('tx_hash'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  completedAt: timestamp('completed_at'),
  failureReason: text('failure_reason'),
});

export const walletCreationRequests = pgTable('wallet_creation_requests', {
  id: serial('id').primaryKey(),
  requestId: text('request_id').notNull().unique(),
  userId: integer('user_id').references(() => users.id).notNull(),
  status: walletCreationStatusEnum('status').default('CREATING').notNull(),
  walletId: text('wallet_id'),
  address: text('address'),
  addresses: text('addresses'), // JSON array of chain-address pairs
  errorMessage: text('error_message'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

export type User = typeof users.$inferSelect;
export type LendingPosition = typeof lendingPositions.$inferSelect;
export type NewLendingPosition = typeof lendingPositions.$inferInsert;
export type WithdrawalAddress = typeof withdrawalAddresses.$inferSelect;
export type NewWithdrawalAddress = typeof withdrawalAddresses.$inferInsert;
export type AddressVerification = typeof addressVerifications.$inferSelect;
export type NewAddressVerification = typeof addressVerifications.$inferInsert;
export type Withdrawal = typeof withdrawals.$inferSelect;
export type NewWithdrawal = typeof withdrawals.$inferInsert;
export type WalletCreationRequest = typeof walletCreationRequests.$inferSelect;
export type NewWalletCreationRequest = typeof walletCreationRequests.$inferInsert;
