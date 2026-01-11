import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { ToastProvider, ToastViewport } from '@/components/ui/toast';
import i18n from '@/i18n';
import AccountOpening from './AccountOpening';
import * as React from 'react';

describe('AccountOpening', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
  });

  const renderComponent = () => {
    return render(
      React.createElement(I18nextProvider, { i18n },
        React.createElement(ToastProvider, null,
          React.createElement(AccountOpening),
          React.createElement(ToastViewport)
        )
      )
    );
  };

  it('should render the page title', () => {
    renderComponent();
    expect(screen.getByText('Activate Your Wallet')).toBeInTheDocument();
  });

  it('should render the page description', () => {
    renderComponent();
    expect(screen.getByText(/Create your secure digital asset wallet/)).toBeInTheDocument();
  });

  it('should render the main card with Safeheron title', () => {
    renderComponent();
    expect(screen.getByText('Safeheron Custody Account')).toBeInTheDocument();
  });

  it('should render the security info text', () => {
    renderComponent();
    expect(screen.getByText(/Your assets are protected/)).toBeInTheDocument();
  });

  it('should render the Activate Now button initially', () => {
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    expect(button).toBeInTheDocument();
    expect(button).not.toBeDisabled();
  });

  it('should show loading state when button is clicked', async () => {
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText(/Creating your wallet account/i)).toBeInTheDocument();
    });
  });

  it('should show success state with address after wallet creation', async () => {
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText('Account Activated Successfully')).toBeInTheDocument();
    });
    expect(screen.getByText(/Your Primary Deposit Address/i)).toBeInTheDocument();
  });

  it('should show wallet ID after success', async () => {
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText(/Wallet ID/i)).toBeInTheDocument();
    });
  });

  it('should render three feature cards', () => {
    renderComponent();
    expect(screen.getByText('Bank-Level Security')).toBeInTheDocument();
    expect(screen.getByText('Instant Setup')).toBeInTheDocument();
    expect(screen.getByText('Full Transparency')).toBeInTheDocument();
  });

  it('should be accessible - have proper heading hierarchy', () => {
    renderComponent();
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1).toBeInTheDocument();
  });

  it('should render in English by default', () => {
    renderComponent();
    expect(screen.getByText('Activate Your Wallet')).toBeInTheDocument();
  });

  it('should switch to Chinese when language changes', async () => {
    renderComponent();
    expect(screen.getByText('Activate Your Wallet')).toBeInTheDocument();
    await act(async () => {
      await i18n.changeLanguage('zh');
    });
    render(
      React.createElement(I18nextProvider, { i18n },
        React.createElement(ToastProvider, null,
          React.createElement(AccountOpening),
          React.createElement(ToastViewport)
        )
      )
    );
    expect(screen.getByText('开通资金账户')).toBeInTheDocument();
  });

  it('should have proper button styling classes', () => {
    renderComponent();
    const button = screen.getByRole('button', { name: /Activate Now/i });
    expect(button).toHaveClass('w-full');
  });
});
