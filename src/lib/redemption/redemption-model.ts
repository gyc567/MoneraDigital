export type RedemptionStatus = 'HOLDING' | 'REDEEMED' | 'RENEWED' | 'FAILED';

export interface RedemptionRecord {
  id: string;
  userId: string;
  productId: string;
  principal: number; // principal invested
  apy: number; // annual percentage yield (e.g., 0.07)
  durationDays: number; // term length in days
  status: RedemptionStatus;
  startDate: string; // ISO date string
  endDate?: string; // ISO date string when maturation occurs
  autoRenew: boolean; // whether to auto-renew on maturity
  interestTotal: number; // total interest for the term
  redemptionAmount: number; // principal + interest
  redeemedAt?: string; // timestamp when redeemed
  renewedToOrderId?: string; // if renewed, reference to new order id
}
