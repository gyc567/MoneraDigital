-- ====================================================================
-- Monera Digital - Complete Database Schema
-- Consolidated from: drizzle/schema.ts, src/db/schema.ts, Go migrations
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
  two_factor_last_used_at BIGINT DEFAULT 0,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_two_factor_last_used_at ON users(two_factor_last_used_at) WHERE two_factor_enabled = TRUE;

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

CREATE INDEX IF NOT EXISTS idx_lending_positions_user_id ON lending_positions(user_id);
CREATE INDEX IF NOT EXISTS idx_lending_positions_asset ON lending_positions(asset);
CREATE INDEX IF NOT EXISTS idx_lending_positions_status ON lending_positions(status);

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

CREATE INDEX IF NOT EXISTS idx_withdrawal_addresses_user_id ON withdrawal_addresses(user_id);

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

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);

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

CREATE INDEX IF NOT EXISTS idx_deposits_user_id ON deposits(user_id);

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

CREATE INDEX IF NOT EXISTS idx_account_user_id ON account(user_id);
CREATE INDEX IF NOT EXISTS idx_account_type ON account(type);
CREATE INDEX IF NOT EXISTS idx_account_frozen_balance ON account(user_id, frozen_balance);
CREATE INDEX IF NOT EXISTS idx_account_updated_at ON account(updated_at);
CREATE UNIQUE INDEX IF NOT EXISTS uk_user_type_currency ON account(user_id, type, currency);

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

CREATE INDEX IF NOT EXISTS idx_account_journal_user_id ON account_journal(user_id);
CREATE INDEX IF NOT EXISTS idx_account_journal_account_id ON account_journal(account_id);
CREATE INDEX IF NOT EXISTS idx_account_journal_biz_type ON account_journal(biz_type);
CREATE INDEX IF NOT EXISTS idx_account_journal_ref_id ON account_journal(ref_id);
CREATE INDEX IF NOT EXISTS idx_account_journal_created_at ON account_journal(created_at);

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

CREATE INDEX IF NOT EXISTS idx_wealth_product_currency ON wealth_product(currency);
CREATE INDEX IF NOT EXISTS idx_wealth_product_status ON wealth_product(status);

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

CREATE INDEX IF NOT EXISTS idx_wealth_order_user_id ON wealth_order(user_id);
CREATE INDEX IF NOT EXISTS idx_wealth_order_product_id ON wealth_order(product_id);
CREATE INDEX IF NOT EXISTS idx_wealth_order_end_date ON wealth_order(end_date);
CREATE INDEX IF NOT EXISTS idx_wealth_order_status ON wealth_order(status);
CREATE INDEX IF NOT EXISTS idx_wealth_order_renewed_from ON wealth_order(renewed_from_order_id);

-- 2.3 Wealth Interest Record
CREATE TABLE IF NOT EXISTS wealth_interest_record (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  type SMALLINT NOT NULL,
  date DATE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_wealth_interest_record_order_id ON wealth_interest_record(order_id);
CREATE INDEX IF NOT EXISTS idx_wealth_interest_record_date ON wealth_interest_record(date);

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

CREATE INDEX IF NOT EXISTS idx_idempotency_record_request_id ON idempotency_record(request_id);
CREATE INDEX IF NOT EXISTS idx_idempotency_record_ttl ON idempotency_record(ttl_expire_at);
CREATE UNIQUE INDEX IF NOT EXISTS uk_idempotency ON idempotency_record(user_id, request_id, biz_type);

-- 3.2 Transfer Record
CREATE TABLE IF NOT EXISTS transfer_record (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  transfer_id TEXT NOT NULL,
  from_account_id BIGINT NOT NULL,
  to_account_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  status TEXT DEFAULT 'PENDING' NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_transfer_record_user_id ON transfer_record(user_id);
CREATE INDEX IF NOT EXISTS idx_transfer_record_transfer_id ON transfer_record(transfer_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_transfer_record_transfer_id ON transfer_record(transfer_id);

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

CREATE INDEX IF NOT EXISTS idx_withdrawal_address_whitelist_user_id ON withdrawal_address_whitelist(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_withdrawal_address_whitelist ON withdrawal_address_whitelist(user_id, wallet_address);

-- 3.5 Withdrawal Request
CREATE TABLE IF NOT EXISTS withdrawal_request (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  request_id TEXT NOT NULL,
  status TEXT DEFAULT 'PROCESSING' NOT NULL,
  error_code TEXT,
  error_message TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_withdrawal_request_request_id ON withdrawal_request(request_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_withdrawal_request_request_id ON withdrawal_request(request_id);

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

CREATE INDEX IF NOT EXISTS idx_withdrawal_order_tx_hash ON withdrawal_order(transaction_hash);
CREATE INDEX IF NOT EXISTS idx_withdrawal_order_user_id ON withdrawal_order(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_withdrawal_order_safeheron ON withdrawal_order(safeheron_order_id);

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

CREATE INDEX IF NOT EXISTS idx_withdrawal_freeze_log_user_id ON withdrawal_freeze_log(user_id);

-- 3.8 Withdrawal Verification
CREATE TABLE IF NOT EXISTS withdrawal_verification (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  withdrawal_order_id INTEGER NOT NULL,
  verification_method TEXT NOT NULL,
  verification_code TEXT,
  attempts INTEGER DEFAULT 0,
  max_attempts INTEGER DEFAULT 3,
  verified BOOLEAN DEFAULT FALSE,
  verified_at TIMESTAMP,
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_withdrawal_verification_user_id ON withdrawal_verification(user_id);

-- 3.9 Wallet Creation Requests
CREATE TABLE IF NOT EXISTS wallet_creation_requests (
  id SERIAL PRIMARY KEY,
  request_id TEXT NOT NULL UNIQUE,
  user_id INTEGER NOT NULL REFERENCES users(id),
  product_code VARCHAR(50) DEFAULT '',
  currency VARCHAR(20) DEFAULT '',
  status wallet_creation_status DEFAULT 'CREATING' NOT NULL,
  wallet_id TEXT,
  address TEXT,
  addresses TEXT,
  error_message TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_wallet_creation_requests_user_id ON wallet_creation_requests(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_wallet_requests_user_product_currency ON wallet_creation_requests(user_id, product_code, currency) WHERE status = 'SUCCESS';

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

CREATE UNIQUE INDEX IF NOT EXISTS uk_wealth_product_approval_product_id ON wealth_product_approval(product_id);

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

CREATE INDEX IF NOT EXISTS idx_audit_trail_operator ON audit_trail(operator_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_trail_action ON audit_trail(action, created_at);

-- 4.4 Admin Users
CREATE TABLE IF NOT EXISTS admin_users (
  id SERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
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
  unfrozen_at TIMESTAMP WITH TIME ZONE,
  CONSTRAINT chk_id CHECK (id = 1)
);

-- ====================================================================
-- VIEWS
-- ====================================================================

CREATE OR REPLACE VIEW v_account_available AS
SELECT
  id,
  user_id,
  type,
  currency,
  balance,
  frozen_balance,
  balance - frozen_balance AS available_balance,
  version,
  created_at,
  updated_at
FROM account
WHERE user_id > 0;

-- ====================================================================
-- COMMENTS
-- ====================================================================

COMMENT ON TABLE users IS 'Core user authentication table with 2FA support';
COMMENT ON COLUMN users.two_factor_secret IS 'Encrypted TOTP secret key';
COMMENT ON COLUMN users.two_factor_enabled IS 'Whether 2FA is enabled for the user';
COMMENT ON COLUMN users.two_factor_backup_codes IS 'Encrypted backup codes for 2FA recovery';
COMMENT ON COLUMN users.two_factor_last_used_at IS 'Timestamp of last 2FA usage for replay attack prevention';
COMMENT ON TABLE lending_positions IS 'Legacy lending positions table';
COMMENT ON TABLE withdrawal_addresses IS 'User withdrawal addresses';
COMMENT ON TABLE withdrawals IS 'Withdrawal transaction records';
COMMENT ON TABLE deposits IS 'Deposit transaction records';
COMMENT ON TABLE account IS 'Unified account table for wealth management';
COMMENT ON TABLE wealth_product IS 'Wealth management products';
COMMENT ON TABLE wealth_order IS 'Wealth product purchase orders';
COMMENT ON TABLE wealth_interest_record IS 'Wealth interest accrual and payment records';
COMMENT ON TABLE idempotency_record IS 'Idempotency prevention for API requests';
COMMENT ON TABLE transfer_record IS 'Account transfer records';
COMMENT ON TABLE withdrawal_address_whitelist IS 'Approved withdrawal addresses';
COMMENT ON TABLE withdrawal_request IS 'Withdrawal request tracking';
COMMENT ON TABLE withdrawal_order IS 'Withdrawal execution orders';
COMMENT ON TABLE withdrawal_freeze_log IS 'Withdrawal balance freeze records';
COMMENT ON TABLE withdrawal_verification IS 'Withdrawal verification codes and attempts';
COMMENT ON TABLE wallet_creation_requests IS 'Wallet creation requests with product and currency support';
COMMENT ON COLUMN wallet_creation_requests.product_code IS 'Product code for the wallet (e.g., X_FINANCE)';
COMMENT ON COLUMN wallet_creation_requests.currency IS 'Currency code for the wallet (e.g., TRON, USDT)';
COMMENT ON TABLE wealth_product_approval IS 'Wealth product approval workflow';
COMMENT ON TABLE account_adjustment IS 'Manual account adjustments';
COMMENT ON TABLE audit_trail IS 'Comprehensive audit trail for all operations';
COMMENT ON TABLE admin_users IS 'Admin user accounts';
COMMENT ON TABLE reconciliation_log IS 'Reconciliation check logs';
COMMENT ON TABLE reconciliation_alert_log IS 'Reconciliation alert logs';
COMMENT ON TABLE reconciliation_error_log IS 'Reconciliation error records';
COMMENT ON TABLE manual_review_queue IS 'Manual review queue items';
COMMENT ON TABLE business_freeze_status IS 'Global business freeze status';
COMMENT ON VIEW v_account_available IS 'Available account balance view';

-- ====================================================================
-- END OF SCHEMA
-- ====================================================================
