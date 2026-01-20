CREATE TYPE "public"."address_type" AS ENUM('BTC', 'ETH', 'USDC', 'USDT');--> statement-breakpoint
CREATE TYPE "public"."deposit_status" AS ENUM('PENDING', 'CONFIRMED', 'FAILED');--> statement-breakpoint
CREATE TYPE "public"."lending_status" AS ENUM('ACTIVE', 'COMPLETED', 'TERMINATED');--> statement-breakpoint
CREATE TYPE "public"."wallet_creation_status" AS ENUM('CREATING', 'SUCCESS', 'FAILED');--> statement-breakpoint
CREATE TYPE "public"."withdrawal_status" AS ENUM('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');--> statement-breakpoint
CREATE TABLE "account_adjustment" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"account_id" bigint NOT NULL,
	"adjustment_amount" numeric(65, 30) NOT NULL,
	"reason" text NOT NULL,
	"requested_by" text,
	"requested_at" timestamp with time zone,
	"reviewed_by" text,
	"reviewed_at" timestamp with time zone,
	"approved_by" text,
	"approved_at" timestamp with time zone,
	"status" text DEFAULT 'PENDING',
	"execution_by" text,
	"executed_at" timestamp with time zone,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "account_journal" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"serial_no" text NOT NULL,
	"user_id" bigint NOT NULL,
	"account_id" bigint NOT NULL,
	"amount" numeric(65, 30) NOT NULL,
	"balance_snapshot" numeric(65, 30) NOT NULL,
	"biz_type" text NOT NULL,
	"ref_id" bigint,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "account_journal_serial_no_unique" UNIQUE("serial_no")
);
--> statement-breakpoint
CREATE TABLE "account" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"type" text NOT NULL,
	"currency" text NOT NULL,
	"balance" numeric(65, 30) DEFAULT '0' NOT NULL,
	"frozen_balance" numeric(65, 30) DEFAULT '0' NOT NULL,
	"version" bigint DEFAULT 1 NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "address_verifications" (
	"id" serial PRIMARY KEY NOT NULL,
	"address_id" integer NOT NULL,
	"token" text NOT NULL,
	"expires_at" timestamp NOT NULL,
	"verified_at" timestamp,
	CONSTRAINT "address_verifications_token_unique" UNIQUE("token")
);
--> statement-breakpoint
CREATE TABLE "audit_trail" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"operator_id" text NOT NULL,
	"operator_role" text NOT NULL,
	"action" text NOT NULL,
	"target_id" bigint,
	"target_type" text,
	"old_value" jsonb,
	"new_value" jsonb,
	"reason" text,
	"ip_address" text,
	"status" text,
	"error_message" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "business_freeze_status" (
	"id" integer PRIMARY KEY DEFAULT 1 NOT NULL,
	"is_frozen" boolean DEFAULT false NOT NULL,
	"freeze_reason" text,
	"frozen_at" timestamp with time zone,
	"unfrozen_at" timestamp with time zone
);
--> statement-breakpoint
CREATE TABLE "deposits" (
	"id" serial PRIMARY KEY NOT NULL,
	"user_id" integer NOT NULL,
	"tx_hash" text NOT NULL,
	"amount" numeric(20, 8) NOT NULL,
	"asset" text NOT NULL,
	"chain" text NOT NULL,
	"status" "deposit_status" DEFAULT 'PENDING' NOT NULL,
	"from_address" text,
	"to_address" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"confirmed_at" timestamp,
	CONSTRAINT "deposits_tx_hash_unique" UNIQUE("tx_hash")
);
--> statement-breakpoint
CREATE TABLE "idempotency_record" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"request_id" text NOT NULL,
	"biz_type" text NOT NULL,
	"status" text DEFAULT 'PROCESSING' NOT NULL,
	"result_data" jsonb,
	"error_message" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"completed_at" timestamp with time zone,
	"ttl_expire_at" timestamp with time zone NOT NULL
);
--> statement-breakpoint
CREATE TABLE "lending_positions" (
	"id" serial PRIMARY KEY NOT NULL,
	"user_id" integer NOT NULL,
	"asset" text NOT NULL,
	"amount" numeric(20, 8) NOT NULL,
	"duration_days" integer NOT NULL,
	"apy" numeric(5, 2) NOT NULL,
	"status" "lending_status" DEFAULT 'ACTIVE' NOT NULL,
	"accrued_yield" numeric(20, 8) DEFAULT '0' NOT NULL,
	"start_date" timestamp DEFAULT now() NOT NULL,
	"end_date" timestamp NOT NULL
);
--> statement-breakpoint
CREATE TABLE "manual_review_queue" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"type" text NOT NULL,
	"description" text NOT NULL,
	"severity" text DEFAULT 'WARNING',
	"reviewed_by" text,
	"review_result" text,
	"reviewed_at" timestamp with time zone,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reconciliation_alert_log" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"alert_time" timestamp with time zone NOT NULL,
	"type" text NOT NULL,
	"description" text NOT NULL,
	"user_total" numeric(65, 30),
	"system_balance" numeric(65, 30),
	"difference" numeric(65, 30),
	"status" text DEFAULT 'CRITICAL',
	"resolved_at" timestamp with time zone,
	"resolution_notes" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reconciliation_error_log" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"account_id" bigint NOT NULL,
	"expected_balance" numeric(65, 30),
	"actual_balance" numeric(65, 30),
	"error_type" text NOT NULL,
	"description" text,
	"resolved" boolean DEFAULT false,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reconciliation_log" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"check_time" timestamp with time zone NOT NULL,
	"type" text NOT NULL,
	"user_total" numeric(65, 30),
	"system_balance" numeric(65, 30),
	"difference" numeric(65, 30),
	"status" text DEFAULT 'SUCCESS',
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "transfer_record" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"transfer_id" text NOT NULL,
	"from_account_id" bigint NOT NULL,
	"to_account_id" bigint NOT NULL,
	"amount" numeric(65, 30) NOT NULL,
	"status" text DEFAULT 'PENDING' NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"completed_at" timestamp with time zone,
	CONSTRAINT "transfer_record_transfer_id_unique" UNIQUE("transfer_id")
);
--> statement-breakpoint
CREATE TABLE "users" (
	"id" serial PRIMARY KEY NOT NULL,
	"email" text NOT NULL,
	"password" text NOT NULL,
	"two_factor_secret" text,
	"two_factor_enabled" boolean DEFAULT false NOT NULL,
	"two_factor_backup_codes" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "users_email_unique" UNIQUE("email")
);
--> statement-breakpoint
CREATE TABLE "wallet_creation_requests" (
	"id" serial PRIMARY KEY NOT NULL,
	"request_id" text NOT NULL,
	"user_id" integer NOT NULL,
	"status" "wallet_creation_status" DEFAULT 'CREATING' NOT NULL,
	"wallet_id" text,
	"address" text,
	"addresses" text,
	"error_message" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "wallet_creation_requests_request_id_unique" UNIQUE("request_id")
);
--> statement-breakpoint
CREATE TABLE "wealth_interest_record" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"order_id" bigint NOT NULL,
	"amount" numeric(65, 30) NOT NULL,
	"type" smallint NOT NULL,
	"date" date NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "wealth_order" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"product_id" bigint NOT NULL,
	"amount" numeric(65, 30) NOT NULL,
	"principal_redeemed" numeric(65, 30) DEFAULT '0' NOT NULL,
	"interest_expected" numeric(65, 30) DEFAULT '0' NOT NULL,
	"interest_paid" numeric(65, 30) DEFAULT '0' NOT NULL,
	"interest_accrued" numeric(65, 30) DEFAULT '0' NOT NULL,
	"start_date" date NOT NULL,
	"end_date" date NOT NULL,
	"last_interest_date" date,
	"auto_renew" boolean DEFAULT false NOT NULL,
	"status" smallint DEFAULT 0 NOT NULL,
	"renewed_from_order_id" bigint,
	"renewed_to_order_id" bigint,
	"redeemed_at" timestamp with time zone,
	"redemption_amount" numeric(65, 30),
	"redemption_type" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "wealth_product_approval" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"product_id" bigint NOT NULL,
	"current_step" text DEFAULT 'CREATED',
	"finance_reviewed_by" text,
	"finance_review_at" timestamp with time zone,
	"finance_approved" boolean,
	"finance_comment" text,
	"risk_reviewed_by" text,
	"risk_review_at" timestamp with time zone,
	"risk_approved" boolean,
	"risk_comment" text,
	"admin_approved_by" text,
	"admin_approve_at" timestamp with time zone,
	"admin_approved" boolean,
	"admin_comment" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "wealth_product_approval_product_id_unique" UNIQUE("product_id")
);
--> statement-breakpoint
CREATE TABLE "wealth_product" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"title" text NOT NULL,
	"currency" text NOT NULL,
	"apy" numeric(10, 4) NOT NULL,
	"duration" integer NOT NULL,
	"min_amount" numeric(65, 30) NOT NULL,
	"max_amount" numeric(65, 30) NOT NULL,
	"total_quota" numeric(65, 30) NOT NULL,
	"sold_quota" numeric(65, 30) DEFAULT '0' NOT NULL,
	"status" smallint DEFAULT 1 NOT NULL,
	"auto_renew_allowed" boolean DEFAULT true NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "withdrawal_address_whitelist" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"address_alias" text NOT NULL,
	"chain_type" text NOT NULL,
	"wallet_address" text NOT NULL,
	"verified" boolean DEFAULT false NOT NULL,
	"verified_at" timestamp with time zone,
	"verification_method" text,
	"is_deleted" boolean DEFAULT false NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "withdrawal_addresses" (
	"id" serial PRIMARY KEY NOT NULL,
	"user_id" integer NOT NULL,
	"address" text NOT NULL,
	"address_type" "address_type" NOT NULL,
	"label" text NOT NULL,
	"is_verified" boolean DEFAULT false NOT NULL,
	"is_primary" boolean DEFAULT false NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"verified_at" timestamp,
	"deactivated_at" timestamp
);
--> statement-breakpoint
CREATE TABLE "withdrawal_freeze_log" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"amount" numeric(65, 30) NOT NULL,
	"frozen_at" timestamp with time zone NOT NULL,
	"released_at" timestamp with time zone,
	"reason" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "withdrawal_order" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"amount" numeric(65, 30) NOT NULL,
	"network_fee" numeric(65, 30),
	"platform_fee" numeric(65, 30),
	"actual_amount" numeric(65, 30),
	"chain_type" text NOT NULL,
	"coin_type" text NOT NULL,
	"to_address" text NOT NULL,
	"safeheron_order_id" text,
	"transaction_hash" text,
	"status" text DEFAULT 'PENDING',
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"sent_at" timestamp with time zone,
	"confirmed_at" timestamp with time zone,
	"completed_at" timestamp with time zone,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "withdrawal_request" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"user_id" bigint NOT NULL,
	"request_id" text NOT NULL,
	"status" text DEFAULT 'PROCESSING' NOT NULL,
	"error_code" text,
	"error_message" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "withdrawal_request_request_id_unique" UNIQUE("request_id")
);
--> statement-breakpoint
CREATE TABLE "withdrawals" (
	"id" serial PRIMARY KEY NOT NULL,
	"user_id" integer NOT NULL,
	"from_address_id" integer NOT NULL,
	"amount" numeric(20, 8) NOT NULL,
	"asset" text NOT NULL,
	"to_address" text NOT NULL,
	"status" "withdrawal_status" DEFAULT 'PENDING' NOT NULL,
	"tx_hash" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"completed_at" timestamp,
	"failure_reason" text,
	"fee_amount" numeric(20, 8),
	"received_amount" numeric(20, 8),
	"safeheron_tx_id" text,
	"chain" text
);
--> statement-breakpoint
ALTER TABLE "address_verifications" ADD CONSTRAINT "address_verifications_address_id_withdrawal_addresses_id_fk" FOREIGN KEY ("address_id") REFERENCES "public"."withdrawal_addresses"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "deposits" ADD CONSTRAINT "deposits_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "lending_positions" ADD CONSTRAINT "lending_positions_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "wallet_creation_requests" ADD CONSTRAINT "wallet_creation_requests_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "withdrawal_addresses" ADD CONSTRAINT "withdrawal_addresses_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "withdrawals" ADD CONSTRAINT "withdrawals_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "withdrawals" ADD CONSTRAINT "withdrawals_from_address_id_withdrawal_addresses_id_fk" FOREIGN KEY ("from_address_id") REFERENCES "public"."withdrawal_addresses"("id") ON DELETE no action ON UPDATE no action;