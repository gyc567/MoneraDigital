import { describe, it, expect, vi, beforeEach } from 'vitest';
import { TwoFactorService } from './two-factor-service';
import { db } from './db';
import { authenticator } from 'otplib';
import QRCode from 'qrcode';
import { encrypt, decrypt } from './encryption';

vi.mock('./db');
vi.mock('otplib');
vi.mock('qrcode');
vi.mock('./encryption');

describe('TwoFactorService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('setup', () => {
    it('should generate secret, QR code, and backup codes', async () => {
      const secret = 'test-secret';
      const otpauth = 'otpauth://test';
      const qrCodeUrl = 'data:image/png;base64,qr-code';
      
      (authenticator.generateSecret as any).mockReturnValue(secret);
      (authenticator.keyuri as any).mockReturnValue(otpauth);
      (QRCode.toDataURL as any).mockResolvedValue(qrCodeUrl);
      (encrypt as any).mockImplementation((text: string) => `encrypted-${text}`);
      
      (db as any).update.mockReturnValue({
        set: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([null]),
        }),
      });
      
      const { secret: resultSecret, qrCodeUrl: resultQrCodeUrl, backupCodes } = await TwoFactorService.setup(1, 'test@example.com');

      expect(resultSecret).toBe(secret);
      expect(resultQrCodeUrl).toBe(qrCodeUrl);
      expect(backupCodes).toHaveLength(10);
      expect(db.update).toHaveBeenCalled();
    });
  });

  describe('enable', () => {
    it('should enable 2FA for a user', async () => {
      const user = { twoFactorSecret: 'encrypted-test-secret' };
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as any).mockReturnValue('test-secret');
      (authenticator.check as any).mockReturnValue(true);

      (db as any).update.mockReturnValue({
        set: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([null]),
        }),
      });

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([null]),
      });
      (db as any).update.mockReturnValue({
        set: mockSet,
      });

      const result = await TwoFactorService.enable(1, '123456');

      expect(result).toBe(true);
      expect(db.update).toHaveBeenCalled();
      expect(mockSet).toHaveBeenCalledWith({ twoFactorEnabled: true });
    });

    it('should throw error if 2FA is not set up', async () => {
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([]),
        }),
      });

      await expect(TwoFactorService.enable(1, '123456')).rejects.toThrow('2FA has not been set up');
    });

    it('should throw error for invalid token', async () => {
      const user = { twoFactorSecret: 'encrypted-test-secret' };
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as any).mockReturnValue('test-secret');
      (authenticator.check as any).mockReturnValue(false);

      await expect(TwoFactorService.enable(1, '123456')).rejects.toThrow('Invalid verification code');
    });
  });

  describe('verify', () => {
    it('should return true if 2FA is not enabled', async () => {
      const user = { twoFactorEnabled: false };
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      const result = await TwoFactorService.verify(1, '123456');
      expect(result).toBe(true);
    });

    it('should verify a valid TOTP token', async () => {
      const user = { twoFactorEnabled: true, twoFactorSecret: 'encrypted-test-secret' };
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as any).mockReturnValue('test-secret');
      (authenticator.check as any).mockReturnValue(true);

      const result = await TwoFactorService.verify(1, '123456');
      expect(result).toBe(true);
    });

    it('should verify a valid backup code', async () => {
      const backupCodes = ['ABC-123', 'DEF-456'];
      const user = { 
        twoFactorEnabled: true, 
        twoFactorSecret: 'encrypted-test-secret', 
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`
      };
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as any).mockImplementation((text: string) => text.replace('encrypted-', ''));
      (authenticator.check as any).mockReturnValue(false);

      (db as any).update.mockReturnValue({
        set: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([null]),
        }),
      });

      const result = await TwoFactorService.verify(1, 'ABC-123');
      expect(result).toBe(true);
      expect(db.update).toHaveBeenCalled();
    });

    it('should return false for invalid token and backup code', async () => {
      const backupCodes = ['ABC-123', 'DEF-456'];
      const user = { 
        twoFactorEnabled: true, 
        twoFactorSecret: 'encrypted-test-secret', 
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`
      };
      (db as any).select.mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as any).mockImplementation((text: string) => text.replace('encrypted-', ''));
      (authenticator.check as any).mockReturnValue(false);

      const result = await TwoFactorService.verify(1, 'INVALID');
      expect(result).toBe(false);
    });
  });
});
