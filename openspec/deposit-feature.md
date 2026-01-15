# Feature Proposal: User Deposit System (用户充值系统) - Architect Reviewed

## 1. Overview
Implement a dedicated "Deposit" (充值) feature in the user dashboard to facilitate cryptocurrency deposits (USDT, USDC on various chains). This feature will provide users with their exclusive deposit addresses and real-time status updates on their deposit transactions.

## 2. User Interface (UI)

### 2.1 Sidebar Menu
*   **Action:** Add a new menu item "Deposit" (充值) to the dashboard sidebar.
*   **Icon:** `ArrowDownCircle` or similar.
*   **Position:** Between "Assets" and "Withdraw" (or logical grouping).

### 2.2 Deposit Page (`/dashboard/deposit`)
A new page dedicated to the deposit workflow.

*   **Asset Selection:** Dropdown/Tabs to select Asset (USDT, USDC). Default: USDT.
*   **Network Selection:** Dropdown/Cards to select Network (ETH, TRON, BSC, SOL, etc.). Default: TRON (Low fee).
*   **Address Display:**
    *   Display the specific wallet address for the selected chain (fetched from `/api/wallet/info`).
    *   **Safety:** When switching networks, the address display must clear/fade immediately to prevent copying the wrong address.
    *   QR Code representation.
    *   "Copy" button with feedback toast.
*   **Deposit History Table:**
    *   List recent deposit transactions (fetched from `/api/deposits`).
    *   Columns: Time, Asset, Amount, Chain, TxHash, Status.
    *   **Status Feedback:** Visual indicators for Pending (Yellow), Confirmed (Green), Failed (Red).

## 3. User Experience (UX)

### 3.1 Prerequisite Check (Account Opening)
*   **Check:** On page load, verify wallet status via `/api/wallet/info`.
*   **Logic:**
    *   If `status: SUCCESS`: Render Deposit UI.
    *   If `status: NONE` or `CREATING`: Redirect to `/dashboard/account-opening` or display a "Activate Account" CTA overlay.

### 3.2 Feedback & Notifications
*   **Real-time Updates:** The page should auto-refresh or poll for new deposits.
*   **Success Prompt:** When a deposit changes status to `CONFIRMED` (detected via polling diff), display a prominent success notification (Toast or Sonner) if the user is active on the page.
*   **Failure Prompt:** If a deposit is marked `FAILED` (rare, e.g., compliance rejection), display a clear error message.

## 4. Technical Implementation

### 4.1 Frontend (React)
*   **New Route:** `/dashboard/deposit` -> `src/pages/dashboard/Deposit.tsx`
*   **Sidebar Update:** Update `src/components/DashboardLayout.tsx`.
*   **State Management:**
    *   Use **TanStack Query** (`useQuery`) for fetching wallet info and deposit history.
    *   **Polling:** Enable `refetchInterval: 15000` (15s) for the deposits list to catch incoming transactions.
    *   **Caching:** Cache wallet info (address doesn't change often).

### 4.2 Backend (Go) - *Existing Prerequisites*
*   `GET /api/wallet/info` (Spec: `docs/static-finance-api-wallet-deposit.md`)
    *   Must return map of addresses: `addresses: [{ chain: "ETH", address: "..." }, ...]`
*   `GET /api/deposits` (Spec: `docs/static-finance-api-wallet-deposit.md`)

## 5. Security & Risk Control
*   **Address Integrity:** Frontend must display the address exactly as returned by API. No formatting/truncation in the copy payload.
*   **Network Warning:** When user selects a network (e.g., ERC20), display a warning: "Ensure you are sending via **Ethereum** network. Sending to the wrong network may result in loss of funds."

## 6. Acceptance Criteria
1.  "Deposit" link appears in sidebar.
2.  User is redirected to Account Opening if no wallet exists.
3.  User can view deposit addresses for supported chains (ETH, TRON, etc.).
4.  User can copy address successfully.
5.  Deposit history list automatically updates when funds arrive (within polling interval).
6.  Status changes (Pending -> Success) are visually distinct.