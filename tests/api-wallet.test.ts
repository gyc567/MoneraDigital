import { describe, it, expect, vi, beforeEach } from 'vitest';
import handler from '../api/wallet/[...route]';
import { WalletService } from '../src/lib/wallet-service.js';
import { verifyToken } from '../src/lib/auth-middleware.js';

vi.mock('../src/lib/wallet-service.js');
vi.mock('../src/lib/auth-middleware.js');
vi.mock('../src/lib/rate-limit.js', () => ({
  rateLimit: vi.fn().mockResolvedValue(true)
}));

describe('Wallet Unified Handler', () => {
  let req: any;
  let resObj: any;

  beforeEach(() => {
    vi.clearAllMocks();
    resObj = {
      status: vi.fn().mockReturnThis(),
      json: vi.fn().mockReturnThis(),
    };
  });

  describe('POST /wallet/create', () => {
    it('should create wallet successfully', async () => {
      req = {
        query: { route: ['create'] },
        method: 'POST',
        body: {
          request_id: '550e8400-e29b-41d4-a716-446655440000',
          user_id: 1
        },
        headers: {}
      };
      (verifyToken as any).mockReturnValue({ userId: 1 });

      const mockResult = {
        success: true,
        walletId: 'wallet_sfh_123',
        address: '0x1234567890abcdef',
        status: 'success',
        message: 'Wallet created successfully'
      };
      (WalletService.prototype.createWallet as any).mockResolvedValue(mockResult);

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(200);
      expect(resObj.json).toHaveBeenCalledWith(mockResult);
    });

    it('should return 401 for unauthorized request', async () => {
      req = {
        query: { route: ['create'] },
        method: 'POST',
        body: {
          request_id: '550e8400-e29b-41d4-a716-446655440000',
          user_id: 1
        },
        headers: {}
      };
      (verifyToken as any).mockReturnValue(null);

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(401);
    });

    it('should handle wallet already in progress', async () => {
      req = {
        query: { route: ['create'] },
        method: 'POST',
        body: {
          request_id: '550e8400-e29b-41d4-a716-446655440000',
          user_id: 1
        },
        headers: {}
      };
      (verifyToken as any).mockReturnValue({ userId: 1 });

      const mockResult = {
        success: false,
        status: 'creating',
        message: 'Wallet creation in progress'
      };
      (WalletService.prototype.createWallet as any).mockResolvedValue(mockResult);

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(200);
      expect(resObj.json).toHaveBeenCalledWith(mockResult);
    });
  });

  describe('GET /wallet/status', () => {
    it('should return wallet status', async () => {
      req = {
        query: { route: ['status'] },
        method: 'GET',
        headers: {}
      };
      req.query = { user_id: '1' };
      (verifyToken as any).mockReturnValue({ userId: 1 });

      const mockResult = {
        is_opened: true,
        walletId: 'wallet_sfh_123',
        address: '0x1234567890abcdef',
        status: 'success'
      };
      (WalletService.prototype.getWalletStatus as any).mockResolvedValue(mockResult);

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(200);
      expect(resObj.json).toHaveBeenCalledWith(mockResult);
    });

    it('should return not opened status', async () => {
      req = {
        query: { route: ['status'] },
        method: 'GET',
        headers: {}
      };
      req.query = { user_id: '1' };
      (verifyToken as any).mockReturnValue({ userId: 1 });

      const mockResult = {
        is_opened: false,
        status: 'none'
      };
      (WalletService.prototype.getWalletStatus as any).mockResolvedValue(mockResult);

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(200);
      expect(resObj.json).toHaveBeenCalledWith(mockResult);
    });
  });

  describe('GET /wallet/deposit-address', () => {
    it('should return deposit addresses', async () => {
      req = {
        query: { route: ['deposit-address'] },
        method: 'GET',
        headers: {}
      };
      req.query = { user_id: '1', chain: 'ETH' };
      (verifyToken as any).mockReturnValue({ userId: 1 });

      const mockResult = {
        addresses: [
          { chain: 'ETH', address: '0x1234567890abcdef', memo: null }
        ]
      };
      (WalletService.prototype.getDepositAddresses as any).mockResolvedValue(mockResult);

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(200);
      expect(resObj.json).toHaveBeenCalledWith(mockResult);
    });

    it('should return 404 when no wallet exists', async () => {
      req = {
        query: { route: ['deposit-address'] },
        method: 'GET',
        headers: {}
      };
      req.query = { user_id: '1' };
      (verifyToken as any).mockReturnValue({ userId: 1 });

      (WalletService.prototype.getDepositAddresses as any).mockRejectedValue(
        new Error('No wallet account found for this user')
      );

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(404);
      expect(resObj.json).toHaveBeenCalledWith({ error: 'No wallet account found for this user' });
    });
  });

  describe('Unknown route', () => {
    it('should return 404 for unknown route', async () => {
      req = {
        query: { route: ['unknown'] },
        method: 'POST',
        headers: {}
      };

      await handler(req, resObj);
      expect(resObj.status).toHaveBeenCalledWith(404);
    });
  });
});
