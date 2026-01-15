import { pgTable, serial, text, timestamp, numeric, integer, pgEnum, boolean, bigserial, bigint, jsonb, date, smallint } from 'drizzle-orm/pg-core';

export const lendingStatusEnum = pgEnum('lending_status', ['ACTIVE', 'COMPLETED', 'TERMINATED']);
export const addressTypeEnum = pgEnum('address_type', ['BTC', 'ETH', 'USDC', 'USDT']);
export const withdrawalStatusEnum = pgEnum('withdrawal_status', ['PENDING', 'PROCESSING', 'COMPLETED', 'FAILED']);
export const depositStatusEnum = pgEnum('deposit_status', ['PENDING', 'CONFIRMED', 'FAILED']);
export const walletCreationStatusEnum = pgEnum('wallet_creation_status', ['CREATING', 'SUCCESS', 'FAILED']);

// ====================================================================
// Core User & Auth (Existing)
// ====================================================================

export const users = pgTable('users', {
  id: serial('id').primaryKey(),
  email: text('email').notNull().unique(),
  password: text('password').notNull(),
  twoFactorSecret: text('two_factor_secret'),
  twoFactorEnabled: boolean('two_factor_enabled').default(false).notNull(),
  twoFactorBackupCodes: text('two_factor_backup_codes'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
});

// ====================================================================
// Legacy/Simple Lending (Existing)
// ====================================================================

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

// ====================================================================
// Wallet & Addresses (Existing)
// ====================================================================

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

// ====================================================================
// Transactions (Existing)
// ====================================================================

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
  feeAmount: numeric('fee_amount', { precision: 20, scale: 8 }),
  receivedAmount: numeric('received_amount', { precision: 20, scale: 8 }),
  safeheronTxId: text('safeheron_tx_id'),
  chain: text('chain'),
});

export const deposits = pgTable('deposits', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  txHash: text('tx_hash').notNull().unique(),
  amount: numeric('amount', { precision: 20, scale: 8 }).notNull(),
  asset: text('asset').notNull(),
  chain: text('chain').notNull(),
  status: depositStatusEnum('status').default('PENDING').notNull(),
  fromAddress: text('from_address'),
  toAddress: text('to_address'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  confirmedAt: timestamp('confirmed_at'),
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

// ====================================================================
// New Unified Schema (From SQL Design)
// ====================================================================

// 1.1 Account
export const accounts = pgTable('account', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(), // Assuming simple integer mapping for now, or keep bigint
  type: text('type').notNull(), // FUND, WEALTH
  currency: text('currency').notNull(),
  balance: numeric('balance', { precision: 65, scale: 30 }).default('0').notNull(),
  frozenBalance: numeric('frozen_balance', { precision: 65, scale: 30 }).default('0').notNull(),
  version: bigint('version', { mode: 'number' }).default(1).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// 1.2 Account Journal
export const accountJournals = pgTable('account_journal', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  serialNo: text('serial_no').notNull().unique(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  accountId: bigint('account_id', { mode: 'number' }).notNull(),
  amount: numeric('amount', { precision: 65, scale: 30 }).notNull(),
  balanceSnapshot: numeric('balance_snapshot', { precision: 65, scale: 30 }).notNull(),
  bizType: text('biz_type').notNull(),
  refId: bigint('ref_id', { mode: 'number' }),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 2.1 Wealth Product
export const wealthProducts = pgTable('wealth_product', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  title: text('title').notNull(),
  currency: text('currency').notNull(),
  apy: numeric('apy', { precision: 10, scale: 4 }).notNull(),
  duration: integer('duration').notNull(),
  minAmount: numeric('min_amount', { precision: 65, scale: 30 }).notNull(),
  maxAmount: numeric('max_amount', { precision: 65, scale: 30 }).notNull(),
  totalQuota: numeric('total_quota', { precision: 65, scale: 30 }).notNull(),
  soldQuota: numeric('sold_quota', { precision: 65, scale: 30 }).default('0').notNull(),
  status: smallint('status').default(1).notNull(), // 1-Wait, 2-Open, 3-SoldOut, 4-Closed
  autoRenewAllowed: boolean('auto_renew_allowed').default(true).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// 2.2 Wealth Order
export const wealthOrders = pgTable('wealth_order', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  productId: bigint('product_id', { mode: 'number' }).notNull(),
  amount: numeric('amount', { precision: 65, scale: 30 }).notNull(),
  principalRedeemed: numeric('principal_redeemed', { precision: 65, scale: 30 }).default('0').notNull(),
  interestExpected: numeric('interest_expected', { precision: 65, scale: 30 }).default('0').notNull(),
  interestPaid: numeric('interest_paid', { precision: 65, scale: 30 }).default('0').notNull(),
  interestAccrued: numeric('interest_accrued', { precision: 65, scale: 30 }).default('0').notNull(),
  startDate: date('start_date').notNull(),
  endDate: date('end_date').notNull(),
  lastInterestDate: date('last_interest_date'),
  autoRenew: boolean('auto_renew').default(false).notNull(),
  status: smallint('status').default(0).notNull(),
  renewedFromOrderId: bigint('renewed_from_order_id', { mode: 'number' }),
  renewedToOrderId: bigint('renewed_to_order_id', { mode: 'number' }),
  redeemedAt: timestamp('redeemed_at', { withTimezone: true }),
  redemptionAmount: numeric('redemption_amount', { precision: 65, scale: 30 }),
  redemptionType: text('redemption_type'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// 2.3 Wealth Interest Record
export const wealthInterestRecords = pgTable('wealth_interest_record', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  orderId: bigint('order_id', { mode: 'number' }).notNull(),
  amount: numeric('amount', { precision: 65, scale: 30 }).notNull(),
  type: smallint('type').notNull(), // 1-Accrue, 2-Pay
  date: date('date').notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 3.1 Idempotency Record
export const idempotencyRecords = pgTable('idempotency_record', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  requestId: text('request_id').notNull(),
  bizType: text('biz_type').notNull(),
  status: text('status').default('PROCESSING').notNull(),
  resultData: jsonb('result_data'),
  errorMessage: text('error_message'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  completedAt: timestamp('completed_at', { withTimezone: true }),
  ttlExpireAt: timestamp('ttl_expire_at', { withTimezone: true }).notNull(),
});

// 3.3 Transfer Record
export const transferRecords = pgTable('transfer_record', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  transferId: text('transfer_id').notNull().unique(),
  fromAccountId: bigint('from_account_id', { mode: 'number' }).notNull(),
  toAccountId: bigint('to_account_id', { mode: 'number' }).notNull(),
  amount: numeric('amount', { precision: 65, scale: 30 }).notNull(),
  status: text('status').default('PENDING').notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  completedAt: timestamp('completed_at', { withTimezone: true }),
});

// 3.4 Withdrawal Address Whitelist (Parallel to withdrawalAddresses)
export const withdrawalAddressWhitelists = pgTable('withdrawal_address_whitelist', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  addressAlias: text('address_alias').notNull(),
  chainType: text('chain_type').notNull(),
  walletAddress: text('wallet_address').notNull(),
  verified: boolean('verified').default(false).notNull(),
  verifiedAt: timestamp('verified_at', { withTimezone: true }),
  verificationMethod: text('verification_method'),
  isDeleted: boolean('is_deleted').default(false).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// 3.5 Withdrawal Request
export const withdrawalRequests = pgTable('withdrawal_request', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  requestId: text('request_id').notNull().unique(),
  status: text('status').default('PROCESSING').notNull(),
  errorCode: text('error_code'),
  errorMessage: text('error_message'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 3.6 Withdrawal Order (Parallel to withdrawals)
export const withdrawalOrders = pgTable('withdrawal_order', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  amount: numeric('amount', { precision: 65, scale: 30 }).notNull(),
  networkFee: numeric('network_fee', { precision: 65, scale: 30 }),
  platformFee: numeric('platform_fee', { precision: 65, scale: 30 }),
  actualAmount: numeric('actual_amount', { precision: 65, scale: 30 }),
  chainType: text('chain_type').notNull(),
  coinType: text('coin_type').notNull(),
  toAddress: text('to_address').notNull(),
  safeheronOrderId: text('safeheron_order_id'),
  transactionHash: text('transaction_hash'),
  status: text('status').default('PENDING'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  sentAt: timestamp('sent_at', { withTimezone: true }),
  confirmedAt: timestamp('confirmed_at', { withTimezone: true }),
  completedAt: timestamp('completed_at', { withTimezone: true }),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// 3.7 Withdrawal Freeze Log
export const withdrawalFreezeLogs = pgTable('withdrawal_freeze_log', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  amount: numeric('amount', { precision: 65, scale: 30 }).notNull(),
  frozenAt: timestamp('frozen_at', { withTimezone: true }).notNull(),
  releasedAt: timestamp('released_at', { withTimezone: true }),
  reason: text('reason').notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 4.1 Wealth Product Approval
export const wealthProductApprovals = pgTable('wealth_product_approval', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  productId: bigint('product_id', { mode: 'number' }).notNull().unique(),
  currentStep: text('current_step').default('CREATED'),
  financeReviewedBy: text('finance_reviewed_by'),
  financeReviewAt: timestamp('finance_review_at', { withTimezone: true }),
  financeApproved: boolean('finance_approved'),
  financeComment: text('finance_comment'),
  riskReviewedBy: text('risk_reviewed_by'),
  riskReviewAt: timestamp('risk_review_at', { withTimezone: true }),
  riskApproved: boolean('risk_approved'),
  riskComment: text('risk_comment'),
  adminApprovedBy: text('admin_approved_by'),
  adminApproveAt: timestamp('admin_approve_at', { withTimezone: true }),
  adminApproved: boolean('admin_approved'),
  adminComment: text('admin_comment'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 4.2 Account Adjustment
export const accountAdjustments = pgTable('account_adjustment', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  userId: bigint('user_id', { mode: 'number' }).notNull(),
  accountId: bigint('account_id', { mode: 'number' }).notNull(),
  adjustmentAmount: numeric('adjustment_amount', { precision: 65, scale: 30 }).notNull(),
  reason: text('reason').notNull(),
  requestedBy: text('requested_by'),
  requestedAt: timestamp('requested_at', { withTimezone: true }),
  reviewedBy: text('reviewed_by'),
  reviewedAt: timestamp('reviewed_at', { withTimezone: true }),
  approvedBy: text('approved_by'),
  approvedAt: timestamp('approved_at', { withTimezone: true }),
  status: text('status').default('PENDING'),
  executionBy: text('execution_by'),
  executedAt: timestamp('executed_at', { withTimezone: true }),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 4.3 Audit Trail
export const auditTrails = pgTable('audit_trail', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  operatorId: text('operator_id').notNull(),
  operatorRole: text('operator_role').notNull(),
  action: text('action').notNull(),
  targetId: bigint('target_id', { mode: 'number' }),
  targetType: text('target_type'),
  oldValue: jsonb('old_value'),
  newValue: jsonb('new_value'),
  reason: text('reason'),
  ipAddress: text('ip_address'),
  status: text('status'),
  errorMessage: text('error_message'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 5.1 Reconciliation Log
export const reconciliationLogs = pgTable('reconciliation_log', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  checkTime: timestamp('check_time', { withTimezone: true }).notNull(),
  type: text('type').notNull(),
  userTotal: numeric('user_total', { precision: 65, scale: 30 }),
  systemBalance: numeric('system_balance', { precision: 65, scale: 30 }),
  difference: numeric('difference', { precision: 65, scale: 30 }),
  status: text('status').default('SUCCESS'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 5.2 Reconciliation Alert Log
export const reconciliationAlertLogs = pgTable('reconciliation_alert_log', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  alertTime: timestamp('alert_time', { withTimezone: true }).notNull(),
  type: text('type').notNull(),
  description: text('description').notNull(),
  userTotal: numeric('user_total', { precision: 65, scale: 30 }),
  systemBalance: numeric('system_balance', { precision: 65, scale: 30 }),
  difference: numeric('difference', { precision: 65, scale: 30 }),
  status: text('status').default('CRITICAL'),
  resolvedAt: timestamp('resolved_at', { withTimezone: true }),
  resolutionNotes: text('resolution_notes'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 5.3 Reconciliation Error Log
export const reconciliationErrorLogs = pgTable('reconciliation_error_log', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  accountId: bigint('account_id', { mode: 'number' }).notNull(),
  expectedBalance: numeric('expected_balance', { precision: 65, scale: 30 }),
  actualBalance: numeric('actual_balance', { precision: 65, scale: 30 }),
  errorType: text('error_type').notNull(),
  description: text('description'),
  resolved: boolean('resolved').default(false),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 5.4 Manual Review Queue
export const manualReviewQueues = pgTable('manual_review_queue', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  type: text('type').notNull(),
  description: text('description').notNull(),
  severity: text('severity').default('WARNING'),
  reviewedBy: text('reviewed_by'),
  reviewResult: text('review_result'),
  reviewedAt: timestamp('reviewed_at', { withTimezone: true }),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
});

// 5.5 Business Freeze Status
export const businessFreezeStatuses = pgTable('business_freeze_status', {
  id: integer('id').primaryKey().default(1),
  isFrozen: boolean('is_frozen').default(false).notNull(),
  freezeReason: text('freeze_reason'),
  frozenAt: timestamp('frozen_at', { withTimezone: true }),
  unfrozenAt: timestamp('unfrozen_at', { withTimezone: true }),
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
export type Deposit = typeof deposits.$inferSelect;
export type NewDeposit = typeof deposits.$inferInsert;
export type WalletCreationRequest = typeof walletCreationRequests.$inferSelect;
export type NewWalletCreationRequest = typeof walletCreationRequests.$inferInsert;

// Types for new tables
export type Account = typeof accounts.$inferSelect;
export type AccountJournal = typeof accountJournals.$inferSelect;
export type WealthProduct = typeof wealthProducts.$inferSelect;
export type WealthOrder = typeof wealthOrders.$inferSelect;
export type WealthInterestRecord = typeof wealthInterestRecords.$inferSelect;
export type IdempotencyRecord = typeof idempotencyRecords.$inferSelect;
export type TransferRecord = typeof transferRecords.$inferSelect;
export type WithdrawalAddressWhitelist = typeof withdrawalAddressWhitelists.$inferSelect;
export type WithdrawalRequest = typeof withdrawalRequests.$inferSelect;
export type WithdrawalOrder = typeof withdrawalOrders.$inferSelect;
export type WithdrawalFreezeLog = typeof withdrawalFreezeLogs.$inferSelect;
export type WealthProductApproval = typeof wealthProductApprovals.$inferSelect;
export type AccountAdjustment = typeof accountAdjustments.$inferSelect;
export type AuditTrail = typeof auditTrails.$inferSelect;
export type ReconciliationLog = typeof reconciliationLogs.$inferSelect;
export type ReconciliationAlertLog = typeof reconciliationAlertLogs.$inferSelect;
export type ReconciliationErrorLog = typeof reconciliationErrorLogs.$inferSelect;
export type ManualReviewQueue = typeof manualReviewQueues.$inferSelect;
export type BusinessFreezeStatus = typeof businessFreezeStatuses.$inferSelect;