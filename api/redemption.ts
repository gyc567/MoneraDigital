import type { VercelRequest, VercelResponse } from '@vercel/node';
import { InMemoryRedemptionRepository } from '../src/lib/redemption/redemption-repository';
import { RedemptionService } from '../src/lib/redemption/redemption-service';
import { getProduct } from '../src/lib/redemption/products';

const repo = new InMemoryRedemptionRepository();
const service = new RedemptionService(repo);

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method === 'POST') {
    const { userId, productId, principal, autoRenew } = req.body as any;
    if (!userId || !productId || typeof principal !== 'number') {
      return res.status(400).json({ error: 'Missing required fields' });
    }
    // Basic validation against product catalog
    const product = getProduct(productId);
    if (!product) {
      return res.status(400).json({ error: 'Invalid productId' });
    }
    try {
      const record = await service.createRedemption(userId, productId, principal, !!autoRenew);
      return res.status(201).json({ id: record.id, status: record.status, redemptionAmount: record.redemptionAmount, startDate: record.startDate, endDate: record.endDate });
    } catch (e: any) {
      return res.status(400).json({ error: e?.message ?? 'Error creating redemption' });
    }
  } else if (req.method === 'GET') {
    const { id } = req.query as any;
    if (!id) return res.status(400).json({ error: 'Missing id' });
    try {
      const rec = await service.getRedemption(id);
      return res.status(200).json(rec);
    } catch (e: any) {
      return res.status(404).json({ error: e?.message ?? 'Redemption not found' });
    }
  }
  res.status(405).end();
}
