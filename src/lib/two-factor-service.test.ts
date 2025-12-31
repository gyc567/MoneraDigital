import { describe, it, expect, vi, beforeEach } from 'vitest';
import { TwoFactorService } from './two-factor-service.js';
import { authenticator } from 'otplib';

// Mock DB
vi.mock('./db.js', () => ({
  db: {
    update: vi.fn(() => ({
      set: vi.fn(() => ({
        where: vi.fn(() => Promise.resolve([{ id: 1 }]))
      }))
    })),
    select: vi.fn(() => ({
      from: vi.fn(() => ({
        where: vi.fn(() => Promise.resolve([{ id: 1, twoFactorSecret: 'MOCKSECRET', twoFactorEnabled: true }]))
      }))
    }))
  }
}));

describe('TwoFactorService', () => {
  it('should generate a secret and QR code URL', async () => {
    const result = await TwoFactorService.setup(1, 'test@example.com');
    expect(result.secret).toBeDefined();
    expect(result.qrCodeUrl).toContain('data:image/png;base64');
  });

  it('should verify a correct token', async () => {
    const secret = authenticator.generateSecret();
    const token = authenticator.generate(secret);
    
    // 我们需要临时重写 DB mock 以使用真实的 secret
    const { db } = await import('./db.js');
    (db.select as any).mockReturnValue({
      from: vi.fn(() => ({
        where: vi.fn(() => Promise.resolve([{ id: 1, twoFactorSecret: secret, twoFactorEnabled: true }]))
      }))
    });

    const isValid = await TwoFactorService.verify(1, token);
    expect(isValid).toBe(true);
  });

  it('should reject an incorrect token', async () => {
    const isValid = await TwoFactorService.verify(1, '000000');
    expect(isValid).toBe(false);
  });
});
