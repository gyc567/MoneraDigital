import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ToastProvider, ToastViewport } from '@/components/ui/toast';
import { BrowserRouter } from 'react-router-dom';
import i18n from '@/i18n';
import AccountOpening, { parseAvailableNetworks, getDisplayAddress } from './AccountOpening';
import '@testing-library/jest-dom';
import * as React from 'react';
import { vi } from 'vitest';
import { apiRequest } from '@/lib/api-client';

const queryClient = new QueryClient();

const mockApiRequest = vi.fn();

vi.mock('@/lib/api-client', () => ({
  apiRequest: (...args: unknown[]) => mockApiRequest(...args),
}));

Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: vi.fn((key: string) => {
      if (key === 'token') {
        return 'mock-token';
      }
      return null;
    }),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  },
  writable: true,
});

vi.mock('react-router-dom', () => ({
  BrowserRouter: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useNavigate: vi.fn(),
  useLocation: vi.fn(() => ({ pathname: '/dashboard/account-opening' })),
}));

describe('AccountOpening', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
    queryClient.clear();
    vi.clearAllMocks();
  });

  const renderComponent = () => {
    return render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <ToastProvider>
              <AccountOpening />
              <ToastViewport />
            </ToastProvider>
          </I18nextProvider>
        </QueryClientProvider>
      </BrowserRouter>
    );
  };

  it('should render the page title', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText('Activate Your Wallet')).toBeInTheDocument();
  });

  it('should render the page description', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText(/Create your secure digital asset wallet/)).toBeInTheDocument();
  });

  it('should render the main card with MoneraDigital custody title', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText('MoneraDigital Custody Account')).toBeInTheDocument();
  });

  it('should render the security info text', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText(/Your assets are protected/)).toBeInTheDocument();
  });

  it('should render the Activate Now button initially', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    expect(button).toBeInTheDocument();
    expect(button).not.toBeDisabled();
  });

  it('should show loading state when button is clicked', async () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    mockApiRequest.mockResolvedValueOnce({ status: 'CREATING' });
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText(/Creating your wallet account/i)).toBeInTheDocument();
    });
  });

  it('should show success state with address after wallet creation', async () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    mockApiRequest.mockResolvedValueOnce({
      status: 'SUCCESS',
      walletId: { String: 'wallet_test123', Valid: true },
      addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
    });
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText('账户开通成功')).toBeInTheDocument();
    });
    expect(screen.getByText(/Your Primary Deposit Address/i)).toBeInTheDocument();
  });

  it('should show wallet ID after success', async () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    mockApiRequest.mockResolvedValueOnce({
      status: 'SUCCESS',
      walletId: { String: 'wallet_test123', Valid: true },
      addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
    });
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText(/Wallet ID/i)).toBeInTheDocument();
    });
  });

  it('should render three feature cards', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText('Bank-Level Security')).toBeInTheDocument();
    expect(screen.getByText('Instant Setup')).toBeInTheDocument();
    expect(screen.getByText('Full Transparency')).toBeInTheDocument();
  });

  it('should be accessible - have proper heading hierarchy', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1).toBeInTheDocument();
  });

  it('should render in English by default', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText('Activate Your Wallet')).toBeInTheDocument();
  });

  it('should switch to Chinese when language changes', async () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    expect(screen.getByText('Activate Your Wallet')).toBeInTheDocument();
    await act(async () => {
      await i18n.changeLanguage('zh');
    });

    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <ToastProvider>
              <AccountOpening />
              <ToastViewport />
            </ToastProvider>
          </I18nextProvider>
        </QueryClientProvider>
      </BrowserRouter>
    );
    expect(screen.getByText('开通资金账户')).toBeInTheDocument();
  });

  it('should have proper button styling classes', () => {
    mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    expect(button).toHaveClass('w-full');
  });

  describe('Network Display', () => {
    it('should parse available networks from addresses JSON', () => {
      const addresses = '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW","ETH":"0x71C7656EC7ab88b098defB751B7401B5f6d8976F"}';
      const networks = parseAvailableNetworks(addresses);
      expect(networks).toHaveLength(2);
      expect(networks).toContainEqual({ value: 'TRON', label: 'TRON' });
      expect(networks).toContainEqual({ value: 'ETH', label: 'ETH' });
    });

    it('should return empty array for empty addresses', () => {
      expect(parseAvailableNetworks('')).toEqual([]);
      expect(parseAvailableNetworks(null as unknown as string)).toEqual([]);
    });

    it('should return empty array for invalid JSON', () => {
      expect(parseAvailableNetworks('invalid json')).toEqual([]);
    });

    it('should get display address for selected network', () => {
      const addresses = '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW","ETH":"0x71C7656EC7ab88b098defB751B7401B5f6d8976F"}';
      expect(getDisplayAddress(addresses, 'TRON')).toBe('TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW');
      expect(getDisplayAddress(addresses, 'ETH')).toBe('0x71C7656EC7ab88b098defB751B7401B5f6d8976F');
      expect(getDisplayAddress(addresses, 'BSC')).toBe('');
    });

    it('should return empty string for empty addresses', () => {
      expect(getDisplayAddress('', 'TRON')).toBe('');
      expect(getDisplayAddress(null as unknown as string, 'TRON')).toBe('');
    });

    it('should show network label when single network available', async () => {
      mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
      mockApiRequest.mockResolvedValueOnce({
        status: 'SUCCESS',
        walletId: { String: 'wallet_test123', Valid: true },
        addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
      });

      renderComponent();
      const button = screen.getByRole('button', { name: /Activate Now/i });
      fireEvent.click(button);

      await waitFor(() => {
        expect(screen.getByText('账户开通成功')).toBeInTheDocument();
      });

      // Single network should show static label, not tabs
      expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
      expect(screen.getByText('TRON')).toBeInTheDocument();
    });
  });

  // Add New Address tests temporarily disabled
  // describe('Add New Address', () => {
  //   it('should render Add New Address button in SUCCESS state', async () => {
  //     mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
  //     mockApiRequest.mockResolvedValueOnce({
  //       status: 'SUCCESS',
  //       walletId: { String: 'wallet_test123', Valid: true },
  //       addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
  //     });
  //
  //     renderComponent();
  //     const button = screen.getByRole('button', { name: /Activate Now/i });
  //     fireEvent.click(button);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('账户开通成功')).toBeInTheDocument();
  //     });
  //
  //     expect(screen.getByRole('button', { name: /Add New Address/i })).toBeInTheDocument();
  //   });
  //
  //   it('should open dialog when Add New Address button is clicked', async () => {
  //     mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
  //     mockApiRequest.mockResolvedValueOnce({
  //       status: 'SUCCESS',
  //       walletId: { String: 'wallet_test123', Valid: true },
  //       addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
  //     });
  //
  //     renderComponent();
  //     const activateButton = screen.getByRole('button', { name: /Activate Now/i });
  //     fireEvent.click(activateButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('账户开通成功')).toBeInTheDocument();
  //     });
  //
  //     const addButton = screen.getByRole('button', { name: /Add New Address/i });
  //     fireEvent.click(addButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('添加钱包地址')).toBeInTheDocument();
  //     });
  //
  //     expect(screen.getByText('USDT')).toBeInTheDocument();
  //     expect(screen.getByText('TRON (TRC20)')).toBeInTheDocument();
  //   });
  //
  //   it('should close dialog when cancel button is clicked', async () => {
  //     mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
  //     mockApiRequest.mockResolvedValueOnce({
  //       status: 'SUCCESS',
  //       walletId: { String: 'wallet_test123', Valid: true },
  //       addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
  //     });
  //
  //     renderComponent();
  //     const activateButton = screen.getByRole('button', { name: /Activate Now/i });
  //     fireEvent.click(activateButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('账户开通成功')).toBeInTheDocument();
  //     });
  //
  //     const addButton = screen.getByRole('button', { name: /Add New Address/i });
  //     fireEvent.click(addButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('添加钱包地址')).toBeInTheDocument();
  //     });
  //
  //     const cancelButton = screen.getByRole('button', { name: /取消/i });
  //     fireEvent.click(cancelButton);
  //
  //     await waitFor(() => {
  //       expect(screen.queryByText('USDT')).not.toBeInTheDocument();
  //     });
  //   });
  //
  //   it('should call API with selected token and network on confirm', async () => {
  //     mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
  //     mockApiRequest.mockResolvedValueOnce({
  //       status: 'SUCCESS',
  //       walletId: { String: 'wallet_test123', Valid: true },
  //       addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
  //     });
  //     mockApiRequest.mockResolvedValueOnce({ success: true });
  //
  //     renderComponent();
  //     const activateButton = screen.getByRole('button', { name: /Activate Now/i });
  //     fireEvent.click(activateButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('账户开通成功')).toBeInTheDocument();
  //     });
  //
  //     const addButton = screen.getByRole('button', { name: /Add New Address/i });
  //     fireEvent.click(addButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('添加钱包地址')).toBeInTheDocument();
  //     });
  //
  //     const confirmButton = screen.getByRole('button', { name: /确认添加/i });
  //     fireEvent.click(confirmButton);
  //
  //     await waitFor(() => {
  //       expect(mockApiRequest).toHaveBeenCalledWith('/api/wallet/addresses', expect.objectContaining({
  //         method: 'POST',
  //         body: JSON.stringify({ chain: 'TRON', token: 'USDT' }),
  //       }));
  //     });
  //   });
  //
  //   it('should allow selecting different token', async () => {
  //     mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
  //     mockApiRequest.mockResolvedValueOnce({
  //       status: 'SUCCESS',
  //       walletId: { String: 'wallet_test123', Valid: true },
  //       addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
  //     });
  //     mockApiRequest.mockResolvedValueOnce({ success: true });
  //
  //     renderComponent();
  //     const activateButton = screen.getByRole('button', { name: /Activate Now/i });
  //     fireEvent.click(activateButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('账户开通成功')).toBeInTheDocument();
  //     });
  //
  //     const addButton = screen.getByRole('button', { name: /Add New Address/i });
  //     fireEvent.click(addButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('添加钱包地址')).toBeInTheDocument();
  //     });
  //
  //     const usdtSelect = screen.getByRole('combobox', { name: '' });
  //     fireEvent.mouseDown(usdtSelect);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('USDC')).toBeInTheDocument();
  //     });
  //   });
  //
  //   it('should allow selecting different network', async () => {
  //     mockApiRequest.mockResolvedValueOnce({ status: 'NOT_CREATED' });
  //     mockApiRequest.mockResolvedValueOnce({
  //       status: 'SUCCESS',
  //       walletId: { String: 'wallet_test123', Valid: true },
  //       addresses: { String: '{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}', Valid: true }
  //     });
  //     mockApiRequest.mockResolvedValueOnce({ success: true });
  //
  //     renderComponent();
  //     const activateButton = screen.getByRole('button', { name: /Activate Now/i });
  //     fireEvent.click(activateButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('账户开通成功')).toBeInTheDocument();
  //     });
  //
  //     const addButton = screen.getByRole('button', { name: /Add New Address/i });
  //     fireEvent.click(addButton);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('添加钱包地址')).toBeInTheDocument();
  //     });
  //
  //     const tronSelect = screen.getByRole('combobox', { name: '' });
  //     fireEvent.mouseDown(tronSelect);
  //
  //     await waitFor(() => {
  //       expect(screen.getByText('Ethereum (ERC20)')).toBeInTheDocument();
  //       expect(screen.getByText('BNB Smart Chain (BEP20)')).toBeInTheDocument();
  //     });
  //   });
  // });
});
