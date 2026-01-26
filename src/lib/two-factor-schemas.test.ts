import { describe, it, expect } from 'vitest';
import {
  TwoFactorSetupRequestSchema,
  TwoFactorSetupResponseSchema,
  TwoFactorEnableRequestSchema,
  TwoFactorDisableRequestSchema,
  TwoFactorVerifyLoginRequestSchema,
  TwoFactorStatusRequestSchema,
  TwoFactorStatusResponseSchema,
  LoginResponseSchema,
} from './two-factor-schemas';

describe('2FA Validation Schemas', () => {
  describe('TwoFactorSetupRequestSchema', () => {
    it('should accept empty object (no body required)', () => {
      const result = TwoFactorSetupRequestSchema.safeParse({});
      expect(result.success).toBe(true);
    });

    it('should reject object with extra fields (strict mode)', () => {
      const result = TwoFactorSetupRequestSchema.safeParse({ extra: 'field' });
      expect(result.success).toBe(false);
    });
  });

  describe('TwoFactorSetupResponseSchema', () => {
    it('should validate correct response with all fields', () => {
      const response = {
        secret: 'JBSWY3DPEBLW64TMMQ======', // min 16 chars
        otpauth: 'otpauth://totp/Monera%20Digital:user@example.com?secret=JBSWY3DPEBLW64TMMQ%3D%3D%3D%3D%3D%3D',
        qrCodeUrl: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
        backupCodes: Array(10).fill('ABCD1234'),
      };
      const result = TwoFactorSetupResponseSchema.safeParse(response);
      expect(result.success).toBe(true);
    });

    it('should reject secret shorter than 16 characters', () => {
      const response = {
        secret: 'SHORT', // < 16 chars
        otpauth: 'otpauth://totp/test',
        qrCodeUrl: 'data:image/png;base64,test',
        backupCodes: Array(10).fill('ABCD1234'),
      };
      const result = TwoFactorSetupResponseSchema.safeParse(response);
      expect(result.success).toBe(false);
    });

    it('should reject invalid otpauth URI', () => {
      const response = {
        secret: 'JBSWY3DPEBLW64TMMQ======',
        otpauth: 'not-a-uri', // Not a valid URL
        qrCodeUrl: 'data:image/png;base64,test',
        backupCodes: Array(10).fill('ABCD1234'),
      };
      const result = TwoFactorSetupResponseSchema.safeParse(response);
      expect(result.success).toBe(false);
    });

    it('should reject qrCodeUrl not starting with data:image', () => {
      const response = {
        secret: 'JBSWY3DPEBLW64TMMQ======',
        otpauth: 'otpauth://totp/test',
        qrCodeUrl: 'https://example.com/image.png', // Not a data URL
        backupCodes: Array(10).fill('ABCD1234'),
      };
      const result = TwoFactorSetupResponseSchema.safeParse(response);
      expect(result.success).toBe(false);
    });

    it('should reject if backup codes not exactly 10 items', () => {
      const response = {
        secret: 'JBSWY3DPEBLW64TMMQ======',
        otpauth: 'otpauth://totp/test',
        qrCodeUrl: 'data:image/png;base64,test',
        backupCodes: Array(9).fill('ABCD1234'), // Only 9
      };
      const result = TwoFactorSetupResponseSchema.safeParse(response);
      expect(result.success).toBe(false);
    });

    it('should reject if any backup code not exactly 8 characters', () => {
      const response = {
        secret: 'JBSWY3DPEBLW64TMMQ======',
        otpauth: 'otpauth://totp/test',
        qrCodeUrl: 'data:image/png;base64,test',
        backupCodes: ['ABCD1234', 'SHORT', ...Array(8).fill('ABCD1234')], // One is SHORT
      };
      const result = TwoFactorSetupResponseSchema.safeParse(response);
      expect(result.success).toBe(false);
    });
  });

  describe('TwoFactorEnableRequestSchema', () => {
    it('should accept 6-digit token', () => {
      const result = TwoFactorEnableRequestSchema.safeParse({ token: '123456' });
      expect(result.success).toBe(true);
    });

    it('should reject non-6-digit token', () => {
      const result = TwoFactorEnableRequestSchema.safeParse({ token: '12345' }); // 5 digits
      expect(result.success).toBe(false);
    });

    it('should reject token with non-digits', () => {
      const result = TwoFactorEnableRequestSchema.safeParse({ token: '12345a' });
      expect(result.success).toBe(false);
    });

    it('should reject extra fields (strict mode)', () => {
      const result = TwoFactorEnableRequestSchema.safeParse({ token: '123456', extra: 'field' });
      expect(result.success).toBe(false);
    });
  });

  describe('TwoFactorDisableRequestSchema', () => {
    it('should accept 6-digit token', () => {
      const result = TwoFactorDisableRequestSchema.safeParse({ token: '123456' });
      expect(result.success).toBe(true);
    });

    it('should reject non-6-digit token', () => {
      const result = TwoFactorDisableRequestSchema.safeParse({ token: '1234567' }); // 7 digits
      expect(result.success).toBe(false);
    });

    it('should reject extra fields (strict mode)', () => {
      const result = TwoFactorDisableRequestSchema.safeParse({ token: '123456', extra: 'field' });
      expect(result.success).toBe(false);
    });
  });

  describe('TwoFactorVerifyLoginRequestSchema', () => {
    it('should accept token only', () => {
      const result = TwoFactorVerifyLoginRequestSchema.safeParse({ token: '123456' });
      expect(result.success).toBe(true);
    });

    it('should accept token and sessionId', () => {
      const result = TwoFactorVerifyLoginRequestSchema.safeParse({
        token: '123456',
        sessionId: '550e8400-e29b-41d4-a716-446655440000',
      });
      expect(result.success).toBe(true);
    });

    it('should accept 8-char backup code', () => {
      const result = TwoFactorVerifyLoginRequestSchema.safeParse({ token: 'ABCD1234' });
      expect(result.success).toBe(true);
    });

    it('should reject token shorter than 6 chars', () => {
      const result = TwoFactorVerifyLoginRequestSchema.safeParse({ token: '12345' });
      expect(result.success).toBe(false);
    });

    it('should reject missing token', () => {
      const result = TwoFactorVerifyLoginRequestSchema.safeParse({ sessionId: 'uuid' });
      expect(result.success).toBe(false);
    });

    it('should reject extra fields (strict mode)', () => {
      const result = TwoFactorVerifyLoginRequestSchema.safeParse({ token: '123456', extra: 'field' });
      expect(result.success).toBe(false);
    });
  });

  describe('TwoFactorStatusRequestSchema', () => {
    it('should accept empty object', () => {
      const result = TwoFactorStatusRequestSchema.safeParse({});
      expect(result.success).toBe(true);
    });

    it('should reject extra fields (strict mode)', () => {
      const result = TwoFactorStatusRequestSchema.safeParse({ extra: 'field' });
      expect(result.success).toBe(false);
    });
  });

  describe('TwoFactorStatusResponseSchema', () => {
    it('should accept 2FA enabled with backup codes', () => {
      const result = TwoFactorStatusResponseSchema.safeParse({
        enabled: true,
        remainingBackupCodes: 5,
      });
      expect(result.success).toBe(true);
    });

    it('should accept 2FA disabled', () => {
      const result = TwoFactorStatusResponseSchema.safeParse({
        enabled: false,
        remainingBackupCodes: 0,
      });
      expect(result.success).toBe(true);
    });

    it('should reject negative backup codes count', () => {
      const result = TwoFactorStatusResponseSchema.safeParse({
        enabled: true,
        remainingBackupCodes: -1,
      });
      expect(result.success).toBe(false);
    });

    it('should reject non-integer backup codes', () => {
      const result = TwoFactorStatusResponseSchema.safeParse({
        enabled: true,
        remainingBackupCodes: 5.5,
      });
      expect(result.success).toBe(false);
    });

    it('should reject missing enabled field', () => {
      const result = TwoFactorStatusResponseSchema.safeParse({
        remainingBackupCodes: 5,
      });
      expect(result.success).toBe(false);
    });

    it('should reject missing remainingBackupCodes field', () => {
      const result = TwoFactorStatusResponseSchema.safeParse({
        enabled: true,
      });
      expect(result.success).toBe(false);
    });
  });

  describe('LoginResponseSchema', () => {
    it('should accept successful login without 2FA', () => {
      const result = LoginResponseSchema.safeParse({
        success: true,
        token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',
        user: {
          id: 1,
          email: 'user@example.com',
        },
      });
      expect(result.success).toBe(true);
    });

    it('should accept login requiring 2FA', () => {
      const result = LoginResponseSchema.safeParse({
        requires2FA: true,
        sessionId: '550e8400-e29b-41d4-a716-446655440000',
        user: {
          id: 1,
          email: 'user@example.com',
        },
      });
      expect(result.success).toBe(true);
    });

    it('should reject invalid sessionId (not UUID)', () => {
      const result = LoginResponseSchema.safeParse({
        requires2FA: true,
        sessionId: 'not-a-uuid',
        user: {
          id: 1,
          email: 'user@example.com',
        },
      });
      expect(result.success).toBe(false);
    });

    it('should reject invalid email format', () => {
      const result = LoginResponseSchema.safeParse({
        success: true,
        token: 'jwt-token',
        user: {
          id: 1,
          email: 'not-an-email',
        },
      });
      expect(result.success).toBe(false);
    });

    it('should reject empty token', () => {
      const result = LoginResponseSchema.safeParse({
        success: true,
        token: '',
        user: {
          id: 1,
          email: 'user@example.com',
        },
      });
      expect(result.success).toBe(false);
    });

    it('should reject missing user object', () => {
      const result = LoginResponseSchema.safeParse({
        success: true,
        token: 'jwt-token',
      });
      expect(result.success).toBe(false);
    });

    it('should accept either success response or 2FA response', () => {
      const successResponse = LoginResponseSchema.safeParse({
        success: true,
        token: 'jwt-token',
        user: {
          id: 1,
          email: 'user@example.com',
        },
      });
      expect(successResponse.success).toBe(true);

      const twoFAResponse = LoginResponseSchema.safeParse({
        requires2FA: true,
        sessionId: '550e8400-e29b-41d4-a716-446655440000',
        user: {
          id: 1,
          email: 'user@example.com',
        },
      });
      expect(twoFAResponse.success).toBe(true);
    });
  });
});
