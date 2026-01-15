# Redemption Open Spec

Scope: Implement a fixed-term redemption feature for a closed-end financial product, as described in PRD. Focus on high cohesion, low coupling, and test-driven development. No early redemption; support auto-renewal on maturity.

1. Overview
- Product: Fixed-term redemption for a closed-end product (7 days by default, APY 7%).
- Key behaviors: fixed term, daily interest payout (conceptual), maturity redemption, optional auto-renewal with same terms, zero fees.
- Target audience: Backend services, QA, Product managers.

2. Data Model (core entities)
- RedemptionRecord: id, userId, productId, principal, apy, durationDays, status (HOLDING, REDEEMED), startDate, endDate, autoRenew, interestTotal, redemptionAmount, redeemedAt, renewedToOrderId
- Product: id, name, apy, durationDays, autoRenew

3. API Surfaces
- POST /api/redemption
  Request: { userId: string, productId: string, principal: number, autoRenew?: boolean }
  Response: { id, status, redemptionAmount, startDate, endDate }
- GET /api/redemption/:id
  Response: RedemptionRecord projection

4. Lifecycle
- Create Redemption: status HOLDING; compute interestTotal and redemptionAmount
- On maturity: if autoRenew then create new Redemption with renewed principal; otherwise mark as REDEEMED

5. Validation & Errors
- Missing fields -> 400
- Invalid productId -> 400
- Redemption not found -> 404

6. Non-functional
- Tests: unit tests for service; integration-like tests for API path
- No changes to unrelated features

7. Acceptance Criteria
- All unit tests pass; API returns expected fields; maturity flow works as expected; auto-renew creates a new order
