import crypto from "crypto";
import logger from "./logger.js";

// Safeheron API configuration
interface SafeheronConfig {
  apiKey: string;
  apiSecret: string;
  baseUrl: string;
}

const getSafeheronConfig = (): SafeheronConfig => {
  const apiKey = import.meta.env.VITE_SAFEHERON_API_KEY || '';
  const apiSecret = import.meta.env.VITE_SAFEHERON_API_SECRET || '';
  const baseUrl = import.meta.env.VITE_SAFEHERON_API_URL || "https://api.safeheron.vip";

  if (!apiKey || !apiSecret) {
    logger.warn("Safeheron API credentials not configured (VITE_SAFEHERON_*), using mock mode");
    return {
      apiKey: "",
      apiSecret: "",
      baseUrl,
    };
  }

  return { apiKey, apiSecret, baseUrl };
};

// Generate HMAC signature for Safeheron API
const generateSignature = (apiSecret: string, timestamp: string, method: string, path: string, body: string): string => {
  const message = `${timestamp}${method}${path}${body}`;
  const hmac = crypto.createHmac("sha256", apiSecret);
  hmac.update(message);
  return hmac.digest("hex");
};

// Safeheron API response types
interface SafeheronCoinOutResponse {
  txId: string;
  status: "INIT" | "PROCESSING" | "COMPLETED" | "FAILED";
  txHash?: string;
}

interface SafeheronFeeResponse {
  fee: string;
  feeUnit: string;
}

interface SafeheronVault {
  vaultId: string;
  name: string;
  assetId: string;
  balance: string;
}

// Safeheron service class
export class SafeheronService {
  private config: SafeheronConfig;

  constructor() {
    this.config = getSafeheronConfig();
  }

  private isConfigured(): boolean {
    return !!(this.config.apiKey && this.config.apiSecret);
  }

  // Make authenticated request to Safeheron API
  private async request<T>(
    method: string,
    path: string,
    body?: Record<string, unknown>
  ): Promise<T> {
    if (!this.isConfigured()) {
      throw new Error("Safeheron API not configured");
    }

    const timestamp = Date.now().toString();
    const bodyStr = body ? JSON.stringify(body) : "";
    const signature = generateSignature(
      this.config.apiSecret,
      timestamp,
      method,
      path,
      bodyStr
    );

    const headers: Record<string, string> = {
      "X-API-Key": this.config.apiKey,
      "X-Timestamp": timestamp,
      "X-Signature": signature,
      "Content-Type": "application/json",
    };

    const response = await fetch(`${this.config.baseUrl}${path}`, {
      method,
      headers,
      body: bodyStr || undefined,
    });

    if (!response.ok) {
      const errorText = await response.text();
      logger.error(
        { status: response.status, error: errorText, path },
        "Safeheron API request failed"
      );
      throw new Error(`Safeheron API error: ${response.status}`);
    }

    return response.json() as Promise<T>;
  }

  /**
   * Get vault information
   */
  async getVault(vaultId: string): Promise<SafeheronVault | null> {
    if (!this.isConfigured()) {
      logger.info({ vaultId }, "Safeheron not configured, returning mock vault");
      return {
        vaultId,
        name: "Mock Vault",
        assetId: "BTC",
        balance: "1000000",
      };
    }

    try {
      return await this.request<SafeheronVault>("GET", `/v1/vault/${vaultId}`);
    } catch (error: any) {
      logger.error({ error: error.message, vaultId }, "Failed to get vault");
      return null;
    }
  }

  /**
   * Estimate withdrawal fee
   */
  async estimateFee(
    assetId: string,
    amount: string,
    toAddress: string
  ): Promise<SafeheronFeeResponse | null> {
    if (!this.isConfigured()) {
      // Return mock fee based on asset
      const mockFees: Record<string, SafeheronFeeResponse> = {
        BTC: { fee: "0.0005", feeUnit: "BTC" },
        ETH: { fee: "0.002", feeUnit: "ETH" },
        USDC: { fee: "1", feeUnit: "USDC" },
        USDT: { fee: "2", feeUnit: "USDT" },
      };
      return mockFees[assetId] || { fee: "0.001", feeUnit: assetId };
    }

    try {
      return await this.request<SafeheronFeeResponse>(
        "POST",
        "/v1/transaction/estimate-fee",
        { assetId, amount, toAddress }
      );
    } catch (error: any) {
      logger.error(
        { error: error.message, assetId, amount },
        "Failed to estimate fee"
      );
      return null;
    }
  }

  /**
   * Execute Coin Out (withdrawal) via Safeheron
   */
  async coinOut(
    vaultId: string,
    assetId: string,
    amount: string,
    toAddress: string,
    note?: string
  ): Promise<SafeheronCoinOutResponse | null> {
    if (!this.isConfigured()) {
      // Return mock response for development
      logger.info(
        { vaultId, assetId, amount, toAddress },
        "Safeheron not configured, returning mock coin out"
      );
      return {
        txId: `mock_${Date.now()}`,
        status: "INIT",
        txHash: `0x${Math.random().toString(16).substring(2)}`,
      };
    }

    try {
      const response = await this.request<SafeheronCoinOutResponse>(
        "POST",
        "/v1/transaction/coin-out",
        {
          vaultId,
          assetId,
          amount,
          toAddress,
          note: note || `Withdrawal to ${toAddress}`,
        }
      );

      logger.info(
        { txId: response.txId, status: response.status },
        "Coin out initiated"
      );

      return response;
    } catch (error: any) {
      logger.error(
        { error: error.message, vaultId, assetId, amount, toAddress },
        "Failed to execute coin out"
      );
      throw error;
    }
  }

  /**
   * Check transaction status
   */
  async getTransactionStatus(txId: string): Promise<SafeheronCoinOutResponse | null> {
    if (!this.isConfigured()) {
      return {
        txId,
        status: "COMPLETED",
        txHash: `0x${Math.random().toString(16).substring(2)}`,
      };
    }

    try {
      return await this.request<SafeheronCoinOutResponse>(
        "GET",
        `/v1/transaction/${txId}`
      );
    } catch (error: any) {
      logger.error({ error: error.message, txId }, "Failed to get transaction status");
      return null;
    }
  }

  /**
   * Get asset ID mapping
   */
  getAssetId(
    asset: string,
    chain?: string
  ): string {
    const assetMap: Record<string, Record<string, string>> = {
      BTC: { Bitcoin: "BTC" },
      ETH: { Ethereum: "ETH" },
      USDC: {
        Ethereum: "USDC-ERC20",
        Arbitrum: "USDC-ARB",
        Polygon: "USDC-POL",
      },
      USDT: {
        Ethereum: "USDT-ERC20",
        Arbitrum: "USDT-ARB",
        Polygon: "USDT-POL",
        Tron: "USDT-TRC20",
      },
    };

    const chainMap = assetMap[asset];
    if (!chainMap) {
      return asset;
    }

    const selectedChain = chain || "Ethereum";
    return chainMap[selectedChain] || chainMap["Ethereum"] || asset;
  }
}

// Export singleton instance
export const safeheronService = new SafeheronService();

// Export for testing
export type { SafeheronCoinOutResponse, SafeheronFeeResponse, SafeheronVault };
