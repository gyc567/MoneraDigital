import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock modules before importing the service
vi.mock('./db');
vi.mock('otplib');
vi.mock('qrcode');
vi.mock('./encryption', () => ({
  encrypt: vi.fn(),
  decrypt: vi.fn(),
}));
vi.mock('./logger', () => ({
  default: {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
  },
}));

import { TwoFactorService } from './two-factor-service';
import { db } from './db';
import { authenticator } from 'otplib';
import QRCode from 'qrcode';
import { encrypt, decrypt } from './encryption';

describe('TwoFactorService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  // ============================================================
  // setup() tests - 4 test cases
  // ============================================================
  describe('setup', () => {
    it('should generate secret, QR code, and backup codes', async () => {
      const secret = 'test-secret';
      const otpauth = 'otpauth://totp/Monera%20Digital:test@example.com?secret=test-secret';
      const qrCodeUrl = 'data:image/png;base64,qr-code';

      (authenticator.generateSecret as ReturnType<typeof vi.fn>).mockReturnValue(secret);
      (authenticator.keyuri as ReturnType<typeof vi.fn>).mockReturnValue(otpauth);
      (QRCode.toDataURL as ReturnType<typeof vi.fn>).mockResolvedValue(qrCodeUrl);
      (encrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => `encrypted-${text}`);

      (db as any).update = vi.fn().mockReturnValue({
        set: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([{ id: 1 }]),
        }),
      });

      const result = await TwoFactorService.setup(1, 'test@example.com');

      expect(result.secret).toBe(secret);
      expect(result.qrCodeUrl).toBe(qrCodeUrl);
      expect(result.backupCodes).toHaveLength(10);
      expect(result.otpauth).toBe(otpauth);
      expect(db.update).toHaveBeenCalled();
      expect(encrypt).toHaveBeenCalledWith(secret);
    });

    it('should generate unique backup codes each time', async () => {
      const secret = 'test-secret';
      const otpauth = 'otpauth://test';
      const qrCodeUrl = 'data:image/png;base64,qr-code';

      (authenticator.generateSecret as ReturnType<typeof vi.fn>).mockReturnValue(secret);
      (authenticator.keyuri as ReturnType<typeof vi.fn>).mockReturnValue(otpauth);
      (QRCode.toDataURL as ReturnType<typeof vi.fn>).mockResolvedValue(qrCodeUrl);
      (encrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => `encrypted-${text}`);

      (db as any).update = vi.fn().mockReturnValue({
        set: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([{ id: 1 }]),
        }),
      });

      const result1 = await TwoFactorService.setup(1, 'test@example.com');
      const result2 = await TwoFactorService.setup(2, 'test2@example.com');

      // Backup codes should be different between calls (statistically)
      expect(result1.backupCodes).not.toEqual(result2.backupCodes);
    });

    it('should store encrypted secret and backup codes in database', async () => {
      const secret = 'test-secret';
      const otpauth = 'otpauth://test';
      const qrCodeUrl = 'data:image/png;base64,qr-code';

      (authenticator.generateSecret as ReturnType<typeof vi.fn>).mockReturnValue(secret);
      (authenticator.keyuri as ReturnType<typeof vi.fn>).mockReturnValue(otpauth);
      (QRCode.toDataURL as ReturnType<typeof vi.fn>).mockResolvedValue(qrCodeUrl);
      (encrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => `encrypted-${text}`);

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([{ id: 1 }]),
      });
      (db as any).update = vi.fn().mockReturnValue({ set: mockSet });

      await TwoFactorService.setup(1, 'test@example.com');

      expect(mockSet).toHaveBeenCalledWith(
        expect.objectContaining({
          twoFactorSecret: expect.stringContaining('encrypted-'),
          twoFactorBackupCodes: expect.stringContaining('encrypted-'),
        })
      );
    });

    it('should throw error if QR code generation fails', async () => {
      const secret = 'test-secret';
      const otpauth = 'otpauth://test';

      (authenticator.generateSecret as ReturnType<typeof vi.fn>).mockReturnValue(secret);
      (authenticator.keyuri as ReturnType<typeof vi.fn>).mockReturnValue(otpauth);
      (QRCode.toDataURL as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('QR generation failed'));

      await expect(TwoFactorService.setup(1, 'test@example.com')).rejects.toThrow('QR generation failed');
    });
  });

  // ============================================================
  // enable() tests - 4 test cases
  // ============================================================
  describe('enable', () => {
    it('should enable 2FA for a user with valid token', async () => {
      const user = { id: 1, twoFactorSecret: 'encrypted-test-secret', twoFactorEnabled: false };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('test-secret');
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(true);

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([{ id: 1 }]),
      });
      (db as any).update = vi.fn().mockReturnValue({ set: mockSet });

      const result = await TwoFactorService.enable(1, '123456');

      expect(result).toBe(true);
      expect(db.update).toHaveBeenCalled();
      expect(mockSet).toHaveBeenCalledWith({ twoFactorEnabled: true });
    });

    it('should throw error if user not found', async () => {
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([]),
        }),
      });

      await expect(TwoFactorService.enable(1, '123456')).rejects.toThrow('2FA has not been set up');
    });

    it('should throw error if 2FA secret not set up', async () => {
      const user = { id: 1, twoFactorSecret: null, twoFactorEnabled: false };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      await expect(TwoFactorService.enable(1, '123456')).rejects.toThrow('2FA has not been set up');
    });

    it('should throw error for invalid verification token', async () => {
      const user = { id: 1, twoFactorSecret: 'encrypted-test-secret', twoFactorEnabled: false };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('test-secret');
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);

      await expect(TwoFactorService.enable(1, 'wrong-token')).rejects.toThrow('Invalid verification code');
    });
  });

  // ============================================================
  // disable() tests - 4 test cases
  // ============================================================
  describe('disable', () => {
    it('should disable 2FA with valid token', async () => {
      const user = {
        id: 1,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorEnabled: true,
        twoFactorBackupCodes: 'encrypted-backup-codes',
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('test-secret');
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(true);

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([{ id: 1 }]),
      });
      (db as any).update = vi.fn().mockReturnValue({ set: mockSet });

      const result = await TwoFactorService.disable(1, '123456');

      expect(result).toBe(true);
      expect(db.update).toHaveBeenCalled();
      expect(mockSet).toHaveBeenCalledWith({
        twoFactorEnabled: false,
        twoFactorSecret: null,
        twoFactorBackupCodes: null,
      });
    });

    it('should throw error if 2FA is not enabled', async () => {
      const user = {
        id: 1,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorEnabled: false,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      await expect(TwoFactorService.disable(1, '123456')).rejects.toThrow('2FA is not enabled');
    });

    it('should throw error if user not found', async () => {
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([]),
        }),
      });

      await expect(TwoFactorService.disable(999, '123456')).rejects.toThrow('2FA is not enabled');
    });

    it('should throw error for invalid verification token', async () => {
      const user = {
        id: 1,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorEnabled: true,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('test-secret');
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);

      await expect(TwoFactorService.disable(1, 'wrong-token')).rejects.toThrow('Invalid verification code');
    });

    it('should throw error if 2FA enabled but secret is null', async () => {
      const user = {
        id: 1,
        twoFactorSecret: null,
        twoFactorEnabled: true,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      await expect(TwoFactorService.disable(1, '123456')).rejects.toThrow('2FA is not enabled');
    });
  });

  // ============================================================
  // verify() tests - 8 test cases
  // ============================================================
  describe('verify', () => {
    it('should return true if 2FA is not enabled', async () => {
      const user = { id: 1, twoFactorEnabled: false, twoFactorSecret: null };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      const result = await TwoFactorService.verify(1, '123456');
      expect(result).toBe(true);
    });

    it('should return true if user not found (no 2FA required)', async () => {
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([]),
        }),
      });

      const result = await TwoFactorService.verify(999, '123456');
      expect(result).toBe(true);
    });

    it('should return true if secret is not set (no 2FA required)', async () => {
      const user = { id: 1, twoFactorEnabled: true, twoFactorSecret: null };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      const result = await TwoFactorService.verify(1, '123456');
      expect(result).toBe(true);
    });

    it('should verify a valid TOTP token', async () => {
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: null,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('test-secret');
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(true);

      const result = await TwoFactorService.verify(1, '123456');
      expect(result).toBe(true);
    });

    it('should return false for invalid TOTP token without backup codes', async () => {
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: null,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('test-secret');
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);

      const result = await TwoFactorService.verify(1, 'wrong-token');
      expect(result).toBe(false);
    });

    it('should verify a valid backup code', async () => {
      const backupCodes = ['ABCD1234', 'EFGH5678'];
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => text.replace('encrypted-', ''));
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);
      (encrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => `encrypted-${text}`);

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([{ id: 1 }]),
      });
      (db as any).update = vi.fn().mockReturnValue({ set: mockSet });

      const result = await TwoFactorService.verify(1, 'ABCD1234');

      expect(result).toBe(true);
      expect(db.update).toHaveBeenCalled();
    });

    it('should consume backup code after use (one-time use)', async () => {
      const backupCodes = ['ABCD1234', 'EFGH5678'];
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => text.replace('encrypted-', ''));
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);
      (encrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => `encrypted-${text}`);

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([{ id: 1 }]),
      });
      (db as any).update = vi.fn().mockReturnValue({ set: mockSet });

      await TwoFactorService.verify(1, 'ABCD1234');

      // Verify the backup code was removed
      expect(mockSet).toHaveBeenCalledWith(
        expect.objectContaining({
          twoFactorBackupCodes: expect.stringContaining('EFGH5678'),
        })
      );
      // The used code should not be in the updated list
      const setCallArg = mockSet.mock.calls[0][0];
      expect(setCallArg.twoFactorBackupCodes).not.toContain('ABCD1234');
    });

    it('should verify backup code case-insensitively', async () => {
      const backupCodes = ['ABCD1234', 'EFGH5678'];
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => text.replace('encrypted-', ''));
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);
      (encrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => `encrypted-${text}`);

      const mockSet = vi.fn().mockReturnValue({
        where: vi.fn().mockResolvedValue([{ id: 1 }]),
      });
      (db as any).update = vi.fn().mockReturnValue({ set: mockSet });

      // Lowercase input should match uppercase stored code
      const result = await TwoFactorService.verify(1, 'abcd1234');
      expect(result).toBe(true);
    });

    it('should return false for invalid backup code', async () => {
      const backupCodes = ['ABCD1234', 'EFGH5678'];
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => text.replace('encrypted-', ''));
      (authenticator.check as ReturnType<typeof vi.fn>).mockReturnValue(false);

      const result = await TwoFactorService.verify(1, 'INVALID');
      expect(result).toBe(false);
    });
  });

  // ============================================================
  // getStatus() tests - 4 test cases
  // ============================================================
  describe('getStatus', () => {
    it('should return enabled status with remaining backup codes count', async () => {
      const backupCodes = ['ABCD1234', 'EFGH5678', 'IJKL9012'];
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: `encrypted-${JSON.stringify(backupCodes)}`,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockImplementation((text: string) => text.replace('encrypted-', ''));

      const result = await TwoFactorService.getStatus(1);

      expect(result.enabled).toBe(true);
      expect(result.remainingBackupCodes).toBe(3);
    });

    it('should return disabled status when 2FA is not enabled', async () => {
      const user = {
        id: 1,
        twoFactorEnabled: false,
        twoFactorSecret: null,
        twoFactorBackupCodes: null,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      const result = await TwoFactorService.getStatus(1);

      expect(result.enabled).toBe(false);
      expect(result.remainingBackupCodes).toBe(0);
    });

    it('should return disabled status when user not found', async () => {
      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([]),
        }),
      });

      const result = await TwoFactorService.getStatus(999);

      expect(result.enabled).toBe(false);
      expect(result.remainingBackupCodes).toBe(0);
    });

    it('should handle decryption error gracefully and return 0 backup codes', async () => {
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: 'corrupted-data',
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockImplementation(() => {
        throw new Error('Decryption failed');
      });

      const result = await TwoFactorService.getStatus(1);

      expect(result.enabled).toBe(true);
      expect(result.remainingBackupCodes).toBe(0);
    });

    it('should handle invalid JSON in backup codes gracefully', async () => {
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: 'encrypted-invalid-json',
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });
      (decrypt as ReturnType<typeof vi.fn>).mockReturnValue('invalid-json');

      const result = await TwoFactorService.getStatus(1);

      expect(result.enabled).toBe(true);
      expect(result.remainingBackupCodes).toBe(0);
    });

    it('should return 0 backup codes when backup codes field is null', async () => {
      const user = {
        id: 1,
        twoFactorEnabled: true,
        twoFactorSecret: 'encrypted-test-secret',
        twoFactorBackupCodes: null,
      };

      (db as any).select = vi.fn().mockReturnValue({
        from: vi.fn().mockReturnValue({
          where: vi.fn().mockResolvedValue([user]),
        }),
      });

      const result = await TwoFactorService.getStatus(1);

      expect(result.enabled).toBe(true);
      expect(result.remainingBackupCodes).toBe(0);
    });
  });
});
