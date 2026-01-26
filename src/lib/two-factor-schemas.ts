import { z } from 'zod';

/**
 * Regular expression for validating 6-digit TOTP codes.
 * Used for both enable and disable operations.
 */
const SIX_DIGIT_TOKEN_REGEX = /^\d{6}$/;

/**
 * Schema for 2FA setup request.
 * This endpoint requires authentication but no request body.
 * The user ID is extracted from the JWT token.
 */
export const TwoFactorSetupRequestSchema = z.object({}).strict();

/**
 * Schema for 2FA setup response.
 * Returns the TOTP secret, otpauth URL for authenticator apps,
 * a QR code as a data URL, and 10 backup codes.
 */
export const TwoFactorSetupResponseSchema = z.object({
  /** The TOTP secret key (minimum 16 characters for security) */
  secret: z.string().min(16),
  /** The otpauth URL for authenticator apps (e.g., otpauth://totp/...) */
  otpauth: z.string().url(),
  /** The QR code as a data URL (data:image/png;base64,...) */
  qrCodeUrl: z.string().startsWith('data:image'),
  /** Array of exactly 10 backup codes, each 8 characters */
  backupCodes: z.array(z.string().length(8)).length(10),
});

/**
 * Schema for 2FA enable request.
 * Requires a valid 6-digit TOTP token to confirm 2FA activation.
 */
export const TwoFactorEnableRequestSchema = z.object({
  /** The 6-digit TOTP token from authenticator app */
  token: z.string().regex(SIX_DIGIT_TOKEN_REGEX, 'Token must be exactly 6 digits'),
}).strict();

/**
 * Schema for 2FA enable response.
 * Confirms whether 2FA was successfully enabled.
 */
export const TwoFactorEnableResponseSchema = z.object({
  /** Whether the operation was successful */
  success: z.boolean(),
  /** Human-readable message describing the result */
  message: z.string(),
});

/**
 * Schema for 2FA disable request.
 * Requires a valid 6-digit TOTP token to confirm 2FA deactivation.
 */
export const TwoFactorDisableRequestSchema = z.object({
  /** The 6-digit TOTP token from authenticator app */
  token: z.string().regex(SIX_DIGIT_TOKEN_REGEX, 'Token must be exactly 6 digits'),
}).strict();

/**
 * Schema for 2FA disable response.
 * Confirms whether 2FA was successfully disabled.
 */
export const TwoFactorDisableResponseSchema = z.object({
  /** Whether the operation was successful */
  success: z.boolean(),
  /** Human-readable message describing the result */
  message: z.string(),
});

/**
 * Schema for 2FA verify-login request.
 * Used during login flow when 2FA is enabled.
 * Accepts either a 6-digit TOTP token or an 8-character backup code.
 */
export const TwoFactorVerifyLoginRequestSchema = z.object({
  /** The TOTP token (6 digits) or backup code (8 characters) */
  token: z.string().min(6),
  /** Optional session ID for tracking the login flow */
  sessionId: z.string().optional(),
}).strict();

/**
 * Schema for 2FA verify-login response.
 * Returns the authentication result and JWT token on success.
 */
export const TwoFactorVerifyLoginResponseSchema = z.object({
  /** Whether the verification was successful */
  success: z.boolean(),
  /** JWT token for authenticated session (only present on success) */
  token: z.string().optional(),
  /** Human-readable message describing the result */
  message: z.string(),
});

/**
 * Schema for 2FA status request.
 * This is a GET endpoint with no request body.
 * The user ID is extracted from the JWT token.
 */
export const TwoFactorStatusRequestSchema = z.object({}).strict();

/**
 * Schema for 2FA status response.
 * Returns the current 2FA configuration status for the user.
 */
export const TwoFactorStatusResponseSchema = z.object({
  /** Whether 2FA is currently enabled for the user */
  enabled: z.boolean(),
  /** Number of remaining unused backup codes (0-10) */
  remainingBackupCodes: z.number().int().min(0),
});

/**
 * Type exports for TypeScript inference.
 * These types can be used throughout the codebase for type safety.
 */
export type TwoFactorSetupRequest = z.infer<typeof TwoFactorSetupRequestSchema>;
export type TwoFactorSetupResponse = z.infer<typeof TwoFactorSetupResponseSchema>;
export type TwoFactorEnableRequest = z.infer<typeof TwoFactorEnableRequestSchema>;
export type TwoFactorEnableResponse = z.infer<typeof TwoFactorEnableResponseSchema>;
export type TwoFactorDisableRequest = z.infer<typeof TwoFactorDisableRequestSchema>;
export type TwoFactorDisableResponse = z.infer<typeof TwoFactorDisableResponseSchema>;
export type TwoFactorVerifyLoginRequest = z.infer<typeof TwoFactorVerifyLoginRequestSchema>;
export type TwoFactorVerifyLoginResponse = z.infer<typeof TwoFactorVerifyLoginResponseSchema>;
export type TwoFactorStatusRequest = z.infer<typeof TwoFactorStatusRequestSchema>;
export type TwoFactorStatusResponse = z.infer<typeof TwoFactorStatusResponseSchema>;

/**
 * Schema for login response that includes 2FA detection.
 * The response is one of two types:
 * 1. Successful login without 2FA: includes JWT token
 * 2. Requires 2FA verification: includes sessionId for 2FA endpoint
 */
export const LoginResponseSchema = z.union([
  z.object({
    success: z.literal(true),
    token: z.string().min(1, 'Token must not be empty'),
    user: z.object({
      id: z.number(),
      email: z.string().email(),
    }),
    requires2FA: z.literal(false).optional(),
  }),
  z.object({
    requires2FA: z.literal(true),
    sessionId: z.string().uuid('Session ID must be a valid UUID'),
    user: z.object({
      id: z.number(),
      email: z.string().email(),
    }),
    success: z.literal(false).optional(),
  }),
]);

export type LoginResponse = z.infer<typeof LoginResponseSchema>;