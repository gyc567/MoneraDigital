/**
 * Token Manager for Monera Digital
 *
 * Handles JWT token storage, automatic refresh on 401 responses,
 * and concurrent refresh request management.
 */

import logger from './logger.js';

interface TokenPair {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
}

interface TokenManagerState {
  accessToken: string | null;
  refreshToken: string | null;
  isRefreshing: boolean;
  refreshPromise: Promise<TokenPair> | null;
}

class TokenManager {
  private state: TokenManagerState = {
    accessToken: null,
    refreshToken: null,
    isRefreshing: false,
    refreshPromise: null,
  };

  private readonly TOKEN_KEY = 'token';
  private readonly REFRESH_TOKEN_KEY = 'refreshToken';
  private readonly USER_KEY = 'user';

  constructor() {
    this.loadFromStorage();
  }

  private loadFromStorage(): void {
    if (typeof window === 'undefined') return;

    this.state.accessToken = localStorage.getItem(this.TOKEN_KEY);
    this.state.refreshToken = localStorage.getItem(this.REFRESH_TOKEN_KEY);
  }

  getAccessToken(): string | null {
    return this.state.accessToken;
  }

  getRefreshToken(): string | null {
    return this.state.refreshToken;
  }

  setTokens(tokenPair: TokenPair): void {
    this.state.accessToken = tokenPair.accessToken;
    this.state.refreshToken = tokenPair.refreshToken;

    if (typeof window !== 'undefined') {
      localStorage.setItem(this.TOKEN_KEY, tokenPair.accessToken);
      localStorage.setItem(this.REFRESH_TOKEN_KEY, tokenPair.refreshToken);
    }

    logger.info('Tokens updated successfully');
  }

  clearTokens(): void {
    this.state.accessToken = null;
    this.state.refreshToken = null;
    this.state.isRefreshing = false;
    this.state.refreshPromise = null;

    if (typeof window !== 'undefined') {
      localStorage.removeItem(this.TOKEN_KEY);
      localStorage.removeItem(this.REFRESH_TOKEN_KEY);
      localStorage.removeItem(this.USER_KEY);
    }
  }

  async refreshToken(): Promise<TokenPair> {
    if (this.state.isRefreshing && this.state.refreshPromise) {
      return this.state.refreshPromise;
    }

    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      logger.warn('No refresh token available');
      this.clearTokens();
      throw new Error('No refresh token available');
    }

    this.state.isRefreshing = true;

    const refreshPromise = this.performRefresh(refreshToken);
    this.state.refreshPromise = refreshPromise;

    try {
      const tokenPair = await refreshPromise;
      this.setTokens(tokenPair);
      return tokenPair;
    } catch (error) {
      logger.error({ error }, 'Token refresh failed');
      this.clearTokens();
      throw error;
    } finally {
      this.state.isRefreshing = false;
      this.state.refreshPromise = null;
    }
  }

  private async performRefresh(refreshToken: string): Promise<TokenPair> {
    logger.info('Performing token refresh');

    const response = await fetch('/api/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refreshToken }),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      logger.error({ status: response.status, error: errorData }, 'Token refresh failed');
      throw new Error(errorData.error || 'Token refresh failed');
    }

    const data = await response.json();

    return {
      accessToken: data.accessToken || data.token,
      refreshToken: data.refreshToken,
      tokenType: data.tokenType || 'Bearer',
      expiresIn: data.expiresIn || 86400,
    };
  }

  isAuthenticated(): boolean {
    return !!this.state.accessToken;
  }
}

export const tokenManager = new TokenManager();
export type { TokenPair };
