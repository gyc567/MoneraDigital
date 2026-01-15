export interface Product {
  id: string;
  name: string;
  apy: number; // e.g., 0.07 for 7%
  durationDays: number;
  autoRenew?: boolean;
}

const PRODUCTS: Record<string, Product> = {
  'prod-7d': {
    id: 'prod-7d',
    name: '7天固定收益',
    apy: 0.07,
    durationDays: 7,
    autoRenew: true,
  },
  // Additional products can be added later without touching core logic
};

export function getProduct(productId: string): Product | null {
  const p = PRODUCTS[productId];
  return p ?? null;
}
