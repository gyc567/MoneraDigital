# Withdrawal Feature Implementation Proposal

## 1. Overview
This proposal outlines the implementation of the User Withdrawal Feature for Monera Digital, enabling users to withdraw assets to blockchain addresses via Safeheron integration.

## 2. Architecture

### 2.1 Backend (Go)
The backend will leverage the existing Layered Architecture:
- **Handlers:** `internal/handlers` (HTTP/Gin)
- **Services:** `internal/services` (Business Logic)
- **Repositories:** `internal/repository` (Data Access)
- **Models:** `internal/models` (Domain Entities)

### 2.2 Frontend (React)
The frontend `src/pages/dashboard/Withdraw.tsx` will be updated to consume the new Go backend APIs.

## 3. Database Schema

We will add the following tables to PostgreSQL:

```sql
-- 1. Withdrawal Address Whitelist
CREATE TABLE withdrawal_address_whitelist (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  address_alias VARCHAR(255) NOT NULL,
  chain_type VARCHAR(32) NOT NULL,
  wallet_address VARCHAR(255) NOT NULL,
  verified BOOLEAN DEFAULT FALSE,
  verified_at TIMESTAMP,
  verification_method VARCHAR(32),
  is_deleted BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, wallet_address)
);

-- 2. Withdrawal Orders
CREATE TABLE withdrawal_orders (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  amount DECIMAL(32,16) NOT NULL,
  network_fee DECIMAL(32,16),
  platform_fee DECIMAL(32,16),
  actual_amount DECIMAL(32,16),
  chain_type VARCHAR(32) NOT NULL,
  coin_type VARCHAR(32) NOT NULL,
  to_address VARCHAR(255) NOT NULL,
  safeheron_order_id VARCHAR(64),
  transaction_hash VARCHAR(255),
  status VARCHAR(32) DEFAULT 'PENDING', -- PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Note: 'account' table already exists. We need to ensure it handles 'frozen_balance'.
```

## 4. API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| GET | `/api/addresses` | List user's whitelist addresses |
| POST | `/api/addresses` | Add a new address |
| DELETE | `/api/addresses/:id` | Delete (soft) an address |
| GET | `/api/withdrawals` | List withdrawal history |
| POST | `/api/withdrawals` | Create a withdrawal request |
| POST | `/api/withdrawals/fees` | Estimate fees (optional, can be query param) |

## 5. Implementation Plan (TDD)

### Step 1: Domain & Repository Layer
1. Define Models in `internal/models/`.
2. Implement `WithdrawalRepository` and `AddressRepository` with unit tests.

### Step 2: Service Layer
1. Implement `AddressService` (CRUD, Whitelist logic).
2. Implement `WithdrawalService` (Fee calc, Safeheron integration stub, Transaction management).
3. Unit tests for services.

### Step 3: API Layer
1. Implement Handlers in `internal/handlers/`.
2. Register routes in `internal/routes/`.
3. Integration tests.

### Step 4: Frontend Integration
1. Update `Withdraw.tsx` to match API contracts.
2. Ensure Error handling and Loading states are correct.
