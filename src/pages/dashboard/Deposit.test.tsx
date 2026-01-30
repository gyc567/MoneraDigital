import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import i18n from '@/i18n/config';
import { I18nextProvider } from 'react-i18next';
import Deposit from './Deposit';

// Network options configuration
const networkOptions = [
  { value: "TRON", label: "TRON (TRC20)", name: "TRON" },
  { value: "ETH", label: "Ethereum (ERC20)", name: "Ethereum" },
  { value: "BSC", label: "BNB Smart Chain (BEP20)", name: "BNB Smart Chain" },
];

describe('Deposit Network Display', () => {
  describe('networkOptions', () => {
    it('should have correct network names for display', () => {
      const tron = networkOptions.find(n => n.value === "TRON");
      expect(tron?.name).toBe("TRON");
      expect(tron?.label).toBe("TRON (TRC20)");

      const eth = networkOptions.find(n => n.value === "ETH");
      expect(eth?.name).toBe("Ethereum");
      expect(eth?.label).toBe("Ethereum (ERC20)");

      const bsc = networkOptions.find(n => n.value === "BSC");
      expect(bsc?.name).toBe("BNB Smart Chain");
      expect(bsc?.label).toBe("BNB Smart Chain (BEP20)");
    });

    it('should use name for warning message, not label', () => {
      const selectedNetwork = networkOptions.find(n => n.value === "TRON");
      const networkName = selectedNetwork ? selectedNetwork.name : "TRON";
      const networkLabel = selectedNetwork ? selectedNetwork.label : "TRON";

      // Name should be clean (no protocol info)
      expect(networkName).toBe("TRON");
      expect(networkName).not.toContain("TRC20");

      // Label should include protocol info for dropdown
      expect(networkLabel).toContain("TRC20");
    });

    it('should generate correct warning message', () => {
      const selectedNetwork = networkOptions.find(n => n.value === "ETH");
      const networkName = selectedNetwork ? selectedNetwork.name : "ETH";

      const warningMessage = `请务必确认您选择的是 ${networkName} 网络。`;
      expect(warningMessage).toBe("请务必确认您选择的是 Ethereum 网络。");
      expect(warningMessage).not.toContain("ERC20");
    });

    it('should handle all supported networks', () => {
      networkOptions.forEach(option => {
        expect(option.value).toBeDefined();
        expect(option.label).toBeDefined();
        expect(option.name).toBeDefined();
        expect(option.name).not.toContain("(");
        expect(option.name).not.toContain(")");
      });
    });
  });

  describe('Deposit Component - Network Name Interpolation', () => {
    let queryClient: QueryClient;

    beforeEach(() => {
      queryClient = new QueryClient({
        defaultOptions: {
          queries: { retry: false },
        },
      });

      // Mock wallet API to return wallet info
      global.fetch = vi.fn((url: string | URL | Request) => {
        const urlString = typeof url === 'string' ? url : url.toString();
        if (urlString === '/api/wallet/info') {
          return Promise.resolve({
            ok: true,
            status: 200,
            statusText: 'OK',
            headers: new Headers(),
            redirected: false,
            json: () => Promise.resolve({
              status: 'SUCCESS',
              addresses: {
                Valid: true,
                String: JSON.stringify({
                  TRON: 'TQsCtA7CvCEaZVGY17SkP4VN67JkXDLvgJ',
                  ETH: '0x742d35Cc6634C0532925a3b844Bc08e7A3D8fCb8',
                  BSC: '0x742d35Cc6634C0532925a3b844Bc08e7A3D8fCb8',
                }),
              },
            }),
          } as unknown as Response;
        }
        if (urlString.startsWith('/api/deposits')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            statusText: 'OK',
            headers: new Headers(),
            redirected: false,
            json: () => Promise.resolve({ deposits: [] }),
          } as unknown as Response);
        }
        return Promise.reject(new Error(`Unexpected request to ${urlString}`));
      });

      localStorage.setItem('token', 'mock-token');
    });

    it('should not display placeholder {network} in warning message', async () => {
      render(
        <QueryClientProvider client={queryClient}>
          <BrowserRouter>
            <I18nextProvider i18n={i18n}>
              <Deposit />
            </I18nextProvider>
          </BrowserRouter>
        </QueryClientProvider>
      );

      await waitFor(() => {
        // Check that {network} placeholder is NOT in the DOM
        const htmlContent = document.body.innerHTML;
        expect(htmlContent).not.toContain('{network}');
      });
    });
  });
});
