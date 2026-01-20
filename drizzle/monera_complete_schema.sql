-- ====================================================================
-- Monera Digital - Complete Database Schema
-- Generated from drizzle-orm schema
-- Target: PostgreSQL (Neon Database)
-- ====================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ====================================================================
-- ENUMS
-- ====================================================================

CREATE TYPE lending_status AS ENUM ('ACTIVE', 'COMPLETED', 'TERMINATED');
CREATE TYPE address_type AS ENUM ('BTC', 'ETH', 'USDC', 'USDT');
CREATE TYPE withdrawal_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');
CREATE TYPE deposit_status AS ENUM ('PENDING', 'CONFIRMED', 'FAILED');
CREATE TYPE wallet_creation_status AS ENUM ('CREATING', 'SUCCESS', 'FAILED');

-- ====================================================================
-- CORE USER & AUTH
-- ====================================================================

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  two_factor_secret TEXT,
  two_factor_enabled BOOLEAN DEFAULT FALSE NOT NULL,
  two_factor_backup_codes TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

-- ====================================================================
-- LEGACY/SIMPLE LENDING
-- ====================================================================

CREATE TABLE IF NOT EXISTS lending_positions (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  asset TEXT NOT NULL,
  amount NUMERIC(20, 8) NOT NULL,
  duration_days INTEGER NOT NULL,
  apy NUMERIC(5, 2) NOT NULL,
  status lending_status DEFAULT 'ACTIVE' NOT NULL,
  accrued_yield NUMERIC(20, 8) DEFAULT '0' NOT NULL,
  start_date TIMESTAMP DEFAULT NOW() NOT NULL,
  end_date TIMESTAMP NOT NULL
);

-- ====================================================================
-- WALLET & ADDRESSES
-- ====================================================================

CREATE TABLE IF NOT EXISTS withdrawal_addresses (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  address TEXT NOT NULL,
  address_type address_type NOT NULL,
  label TEXT NOT NULL,
  is_verified BOOLEAN DEFAULT FALSE NOT NULL,
  is_primary BOOLEAN DEFAULT FALSE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  verified_at TIMESTAMP,
  deactivated_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS address_verifications (
  id SERIAL PRIMARY KEY,
  address_id INTEGER NOT NULL REFERENCES withdrawal_addresses(id),
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  verified_at TIMESTAMP
);

-- ====================================================================
-- TRANSACTIONS
-- ====================================================================

CREATE TABLE IF NOT EXISTS withdrawals (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  from_address_id INTEGER NOT NULL REFERENCES withdrawal_addresses(id),
  amount NUMERIC(20, 8) NOT NULL,
  asset TEXT NOT NULL,
  to_address TEXT NOT NULL,
  status withdrawal_status DEFAULT 'PENDING' NOT NULL,
  tx_hash TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  completed_at TIMESTAMP,
  failure_reason TEXT,
  fee_amount NUMERIC(20, 8),
  received_amount NUMERIC(20, 8),
  safeheron_tx_id TEXT,
  chain TEXT
);

CREATE TABLE IF NOT EXISTS deposits (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  tx_hash TEXT NOT NULL UNIQUE,
  amount NUMERIC(20, 8) NOT NULL,
  asset TEXT NOT NULL,
  chain TEXT NOT NULL,
  status deposit_status DEFAULT 'PENDING' NOT NULL,
  from_address TEXT,
  to_address TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  confirmed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS wallet_creation_requests (
  id SERIAL PRIMARY KEY,
  request_id TEXT NOT NULL UNIQUE,
  user_id INTEGER NOT NULL REFERENCES users(id),
  status wallet_creation_status DEFAULT 'CREATING' NOT NULL,
  wallet_id TEXT,
  address TEXT,
  addresses TEXT,
  error_message TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

-- ====================================================================
-- NEW UNIFIED SCHEMA (From SQL Design)
-- ====================================================================

-- 1.1 Account
CREATE TABLE IF NOT EXISTS account (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  type TEXT NOT NULL,
  currency TEXT NOT NULL,
  balance NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  frozen_balance NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  version BIGINT DEFAULT 1 NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 1.2 Account Journal
CREATE TABLE IF NOT EXISTS account_journal (
  id BIGSERIAL PRIMARY KEY,
  serial_no TEXT NOT NULL UNIQUE,
  user_id BIGINT NOT NULL,
  account_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  balance_snapshot NUMERIC(65, 30) NOT NULL,
  biz_type TEXT NOT NULL,
  ref_id BIGINT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 2.1 Wealth Product
CREATE TABLE IF NOT EXISTS wealth_product (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  currency TEXT NOT NULL,
  apy NUMERIC(10, 4) NOT NULL,
  duration INTEGER NOT NULL,
  min_amount NUMERIC(65, 30) NOT NULL,
  max_amount NUMERIC(65, 30) NOT NULL,
  total_quota NUMERIC(65, 30) NOT NULL,
  sold_quota NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  status SMALLINT DEFAULT 1 NOT NULL,
  auto_renew_allowed BOOLEAN DEFAULT TRUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 2.2 Wealth Order
CREATE TABLE IF NOT EXISTS wealth_order (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  product_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  principal_redeemed NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  interest_expected NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  interest_paid NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  interest_accrued NUMERIC(65, 30) DEFAULT '0' NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  last_interest_date DATE,
  auto_renew BOOLEAN DEFAULT FALSE NOT NULL,
  status SMALLINT DEFAULT 0 NOT NULL,
  renewed_from_order_id BIGINT,
  renewed_to_order_id BIGINT,
  redeemed_at TIMESTAMP WITH TIME ZONE,
  redemption_amount NUMERIC(65, 30),
  redemption_type TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 2.3 Wealth Interest Record
CREATE TABLE IF NOT EXISTS wealth_interest_record (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  type SMALLINT NOT NULL,
  date DATE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 3.1 Idempotency Record
CREATE TABLE IF NOT EXISTS idempotency_record (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  request_id TEXT NOT NULL,
  biz_type TEXT NOT NULL,
  status TEXT DEFAULT 'PROCESSING' NOT NULL,
  result_data JSONB,
  error_message TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  completed_at TIMESTAMP WITH TIME ZONE,
  ttl_expire_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- 3.3 Transfer Record
CREATE TABLE IF NOT EXISTS transfer_record (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  transfer_id TEXT NOT NULL UNIQUE,
  from_account_id BIGINT NOT NULL,
  to_account_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  status TEXT DEFAULT 'PENDING' NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  completed_at TIMESTAMP WITH TIME ZONE
);

-- 3.4 Withdrawal Address Whitelist
CREATE TABLE IF NOT EXISTS withdrawal_address_whitelist (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  address_alias TEXT NOT NULL,
  chain_type TEXT NOT NULL,
  wallet_address TEXT NOT NULL,
  verified BOOLEAN DEFAULT FALSE NOT NULL,
  verified_at TIMESTAMP WITH TIME ZONE,
  verification_method TEXT,
  is_deleted BOOLEAN DEFAULT FALSE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 3.5 Withdrawal Request
CREATE TABLE IF NOT EXISTS withdrawal_request (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  request_id TEXT NOT NULL UNIQUE,
  status TEXT DEFAULT 'PROCESSING' NOT NULL,
  error_code TEXT,
  error_message TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 3.6 Withdrawal Order
CREATE TABLE IF NOT EXISTS withdrawal_order (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  network_fee NUMERIC(65, 30),
  platform_fee NUMERIC(65, 30),
  actual_amount NUMERIC(65, 30),
  chain_type TEXT NOT NULL,
  coin_type TEXT NOT NULL,
  to_address TEXT NOT NULL,
  safeheron_order_id TEXT,
  transaction_hash TEXT,
  status TEXT DEFAULT 'PENDING',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  sent_at TIMESTAMP WITH TIME ZONE,
  confirmed_at TIMESTAMP WITH TIME ZONE,
  completed_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 3.7 Withdrawal Freeze Log
CREATE TABLE IF NOT EXISTS withdrawal_freeze_log (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  frozen_at TIMESTAMP WITH TIME ZONE NOT NULL,
  released_at TIMESTAMP WITH TIME ZONE,
  reason TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 4.1 Wealth Product Approval
CREATE TABLE IF NOT EXISTS wealth_product_approval (
  id BIGSERIAL PRIMARY KEY,
  product_id BIGINT NOT NULL UNIQUE,
  current_step TEXT DEFAULT 'CREATED',
  finance_reviewed_by TEXT,
  finance_review_at TIMESTAMP WITH TIME ZONE,
  finance_approved BOOLEAN,
  finance_comment TEXT,
  risk_reviewed_by TEXT,
  risk_review_at TIMESTAMP WITH TIME ZONE,
  risk_approved BOOLEAN,
  risk_comment TEXT,
  admin_approved_by TEXT,
  admin_approve_at TIMESTAMP WITH TIME ZONE,
  admin_approved BOOLEAN,
  admin_comment TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 4.2 Account Adjustment
CREATE TABLE IF NOT EXISTS account_adjustment (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  account_id BIGINT NOT NULL,
  adjustment_amount NUMERIC(65, 30) NOT NULL,
  reason TEXT NOT NULL,
  requested_by TEXT,
  requested_at TIMESTAMP WITH TIME ZONE,
  reviewed_by TEXT,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  approved_by TEXT,
  approved_at TIMESTAMP WITH TIME ZONE,
  status TEXT DEFAULT 'PENDING',
  execution_by TEXT,
  executed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 4.3 Audit Trail
CREATE TABLE IF NOT EXISTS audit_trail (
  id BIGSERIAL PRIMARY KEY,
  operator_id TEXT NOT NULL,
  operator_role TEXT NOT NULL,
  action TEXT NOT NULL,
  target_id BIGINT,
  target_type TEXT,
  old_value JSONB,
  new_value JSONB,
  reason TEXT,
  ip_address TEXT,
  status TEXT,
  error_message TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 5.1 Reconciliation Log
CREATE TABLE IF NOT EXISTS reconciliation_log (
  id BIGSERIAL PRIMARY KEY,
  check_time TIMESTAMP WITH TIME ZONE NOT NULL,
  type TEXT NOT NULL,
  user_total NUMERIC(65, 30),
  system_balance NUMERIC(65, 30),
  difference NUMERIC(65, 30),
  status TEXT DEFAULT 'SUCCESS',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 5.2 Reconciliation Alert Log
CREATE TABLE IF NOT EXISTS reconciliation_alert_log (
  id BIGSERIAL PRIMARY KEY,
  alert_time TIMESTAMP WITH TIME ZONE NOT NULL,
  type TEXT NOT NULL,
  description TEXT NOT NULL,
  user_total NUMERIC(65, 30),
  system_balance NUMERIC(65, 30),
  difference NUMERIC(65, 30),
  status TEXT DEFAULT 'CRITICAL',
  resolved_at TIMESTAMP WITH TIME ZONE,
  resolution_notes TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 5.3 Reconciliation Error Log
CREATE TABLE IF NOT EXISTS reconciliation_error_log (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL,
  expected_balance NUMERIC(65, 30),
  actual_balance NUMERIC(65, 30),
  error_type TEXT NOT NULL,
  description TEXT,
  resolved BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 5.4 Manual Review Queue
CREATE TABLE IF NOT EXISTS manual_review_queue (
  id BIGSERIAL PRIMARY KEY,
  type TEXT NOT NULL,
  description TEXT NOT NULL,
  severity TEXT DEFAULT 'WARNING',
  reviewed_by TEXT,
  review_result TEXT,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 5.5 Business Freeze Status
CREATE TABLE IF NOT EXISTS business_freeze_status (
  id INTEGER PRIMARY KEY DEFAULT 1,
  is_frozen BOOLEAN DEFAULT FALSE NOT NULL,
  freeze_reason TEXT,
  frozen_at TIMESTAMP WITH TIME ZONE,
  unfrozen_at TIMESTAMP WITH TIME ZONE
);

-- ====================================================================
-- INDEXES
-- ====================================================================

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_lending_positions_user_id ON lending_positions(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawal_addresses_user_id ON withdrawal_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_deposits_user_id ON deposits(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_creation_requests_user_id ON wallet_creation_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_account_user_id ON account(user_id);
CREATE INDEX IF NOT EXISTS idx_account_journal_user_id ON account_journal(user_id);
CREATE INDEX IF NOT EXISTS idx_wealth_order_user_id ON wealth_order(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawal_order_user_id ON withdrawal_order(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_trail_operator_id ON audit_trail(operator_id);

-- ====================================================================
-- COMMENTS
-- ====================================================================

COMMENT ON TABLE users IS 'Core user authentication table';
COMMENT ON TABLE lending_positions IS 'Legacy lending positions table';
COMMENT ON TABLE withdrawal_addresses IS 'User withdrawal addresses';
COMMENT ON TABLE withdrawals IS 'Withdrawal transaction records';
COMMENT ON TABLE deposits IS 'Deposit transaction records';
COMMENT ON TABLE account IS 'Unified account table for wealth management';
COMMENT ON TABLE wealth_product IS 'Wealth management products';
COMMENT ON TABLE wealth_order IS 'Wealth product purchase orders';
COMMENT ON TABLE audit_trail IS 'Comprehensive audit trail for all operations';

-- ====================================================================
-- END OF SCHEMA
-- ====================================================================
