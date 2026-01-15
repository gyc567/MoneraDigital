import { InMemoryRedemptionRepository } from '../lib/redemption/redemption-repository';
import { RedemptionService } from '../lib/redemption/redemption-service';
import { getProduct } from '../lib/redemption/products';

// Simple test setup using in-memory repo
describe('RedemptionService', () => {
  test('createRedemption computes redemption amount correctly', async () => {
    const repo = new InMemoryRedemptionRepository();
    const svc = new RedemptionService(repo);
    const productId = 'prod-7d';
    const userId = 'user-123';
    const principal = 1000;
    const rec = await svc.createRedemption(userId, productId, principal, false);
    expect(rec.userId).toBe(userId);
    expect(rec.productId).toBe(productId);
    expect(rec.principal).toBe(principal);
    expect(rec.apy).toBe(0.07);
    expect(rec.durationDays).toBe(7);
    const expectedInterest = principal * 0.07 * 7 / 365;
    expect(rec.interestTotal).toBeCloseTo(expectedInterest);
    expect(rec.redemptionAmount).toBeCloseTo(principal + expectedInterest);
    expect(rec.status).toBe('HOLDING');
  });

  test('redeem maturity without autoRenew marks as redeemed', async () => {
    const repo = new InMemoryRedemptionRepository();
    const svc = new RedemptionService(repo);
    const rec = await svc.createRedemption('u1', 'prod-7d', 1000, false);
    const redeemed = await svc.redeemMaturity(rec.id);
    // Ensure operation completed and state updated
    expect(redeemed).toBeDefined();
    expect(redeemed.status).toBe('REDEEMED');
  });

  test('redeem maturity with autoRenew creates a renewed order', async () => {
    const repo = new InMemoryRedemptionRepository();
    const svc = new RedemptionService(repo);
    const rec = await svc.createRedemption('u2', 'prod-7d', 1000, true);
    const renewed = await svc.redeemMaturity(rec.id);
    // Should create a new record and link to renewed order
    expect(renewed).toBeDefined();
    expect(renewed.status).toBe('HOLDING');
  });

  test('getRedemption returns existing record', async () => {
    const repo = new InMemoryRedemptionRepository();
    const svc = new RedemptionService(repo);
    const rec = await svc.createRedemption('u3', 'prod-7d', 1500, false);
    const fetched = await svc.getRedemption(rec.id);
    expect(fetched).toBeDefined();
    expect(fetched.id).toBe(rec.id);
  });
});
