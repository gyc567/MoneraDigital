import { RedemptionRecord } from './redemption-model';

export interface IRedemptionRepository {
  create(record: RedemptionRecord): Promise< RedemptionRecord >;
  get(id: string): Promise< RedemptionRecord | null >;
  update(record: RedemptionRecord): Promise<void>;
  list(): Promise<RedemptionRecord[]>;
}

export class InMemoryRedemptionRepository implements IRedemptionRepository {
  private store: Map<string, RedemptionRecord> = new Map();

  async create(record: RedemptionRecord): Promise<RedemptionRecord> {
    this.store.set(record.id, record);
    return record;
  }

  async get(id: string): Promise<RedemptionRecord | null> {
    return this.store.get(id) ?? null;
  }

  async update(record: RedemptionRecord): Promise<void> {
    if (!this.store.has(record.id)) throw new Error('RedemptionRecord not found');
    this.store.set(record.id, record);
  }

  async list(): Promise<RedemptionRecord[]> {
    return Array.from(this.store.values());
  }
}
