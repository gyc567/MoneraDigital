import { InMemoryRedemptionRepository } from './redemption-repository.js';
import { RedemptionRecord } from './redemption-model.js';
import { getProduct, Product } from './products.js';
import { addDays } from './utils.js';

export class RedemptionService {
  private repo: InMemoryRedemptionRepository;
  constructor(repo?: InMemoryRedemptionRepository) {
    this.repo = repo ?? new InMemoryRedemptionRepository();
  }

  private computeInterest(principal: number, apy: number, durationDays: number): number {
    return principal * apy * durationDays / 365;
  }

  async createRedemption(userId: string, productId: string, principal: number, autoRenew: boolean): Promise<RedemptionRecord> {
    if (principal <= 0) throw new Error('Invalid principal');
    const product: Product | null = getProduct(productId);
    if (!product) throw new Error('Product not found');

    // Use product defaults if duration not provided; here durationDays comes from product
    const durationDays = product.durationDays;
    const apy = product.apy;

    const now = new Date();
    const endDate = addDays(now, durationDays);
    const interestTotal = this.computeInterest(principal, apy, durationDays);
    const redemptionAmount = principal + interestTotal;

    const record: RedemptionRecord = {
      id: 'REDEEM-' + Date.now() + '-' + Math.floor(Math.random() * 1000),
      userId,
      productId,
      principal,
      apy,
      durationDays,
      status: 'HOLDING',
      startDate: now.toISOString(),
      endDate: endDate.toISOString(),
      autoRenew,
      interestTotal,
      redemptionAmount
    };

    await this.repo.create(record);
    return record;
  }

  async redeemMaturity(id: string): Promise<RedemptionRecord> {
    const rec = await this.repo.get(id);
    if (!rec) throw new Error('Redemption not found');
    if (rec.status !== 'HOLDING') throw new Error('Invalid redemption state');

    const now = new Date();
    const product = getProduct(rec.productId);
    if (!product) throw new Error('Product not found');

    // Auto renew logic: if enabled, create a new redemption order with renewed principal
    if (rec.autoRenew) {
      const renewedPrincipal = rec.redemptionAmount;
      const renewedEnd = addDays(now, product.durationDays);
      const renewedInterest = this.computeInterest(renewedPrincipal, product.apy, product.durationDays);
      const renewedAmount = renewedPrincipal + renewedInterest;
      const renewedRecord: RedemptionRecord = {
        id: 'REDEEM-' + Date.now() + '-' + Math.floor(Math.random() * 1000),
        userId: rec.userId,
        productId: rec.productId,
        principal: renewedPrincipal,
        apy: product.apy,
        durationDays: product.durationDays,
        status: 'HOLDING',
        startDate: now.toISOString(),
        endDate: renewedEnd.toISOString(),
        autoRenew: rec.autoRenew,
        interestTotal: renewedInterest,
        redemptionAmount: renewedAmount
      };
      await this.repo.create(renewedRecord);

      rec.status = 'REDEEMED';
      rec.redeemedAt = now.toISOString();
      rec.renewedToOrderId = renewedRecord.id;
      await this.repo.update(rec);
      return renewedRecord;
    } else {
      rec.status = 'REDEEMED';
      rec.redeemedAt = now.toISOString();
      await this.repo.update(rec);
      return rec;
    }
  }

  async getRedemption(id: string): Promise<RedemptionRecord> {
    const rec = await this.repo.get(id);
    if (!rec) throw new Error('Redemption not found');
    return rec;
  }
}
