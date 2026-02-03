import { z } from "zod";
import { safeheronService } from "./safeheron-service.js";
import logger from "./logger.js";

// Fee calculation schema
export const feeCalculationSchema = z.object({
  asset: z.enum(["BTC", "ETH", "USDC", "USDT"]),
  amount: z.string().refine((val) => !isNaN(parseFloat(val)) && parseFloat(val) > 0, "Invalid amount"),
  chain: z.string().optional(),
});

// Fee calculation result
export interface FeeCalculationResult {
  asset: string;
  amount: string;
  chain: string;
  fee: string;
  receivedAmount: string;
  feePercentage: number;
}

// Fee rates by asset and chain (for fallback calculation)
const FEE_RATES: Record<string, Record<string, { fixed: string; percentage: number }>> = {
  BTC: {
    Bitcoin: { fixed: "0.0005", percentage: 0.001 },
  },
  ETH: {
    Ethereum: { fixed: "0.002", percentage: 0.001 },
  },
  USDC: {
    Ethereum: { fixed: "1", percentage: 0.001 },
    Arbitrum: { fixed: "0.1", percentage: 0.001 },
    Polygon: { fixed: "0.1", percentage: 0.001 },
  },
  USDT: {
    Ethereum: { fixed: "2", percentage: 0.001 },
    Arbitrum: { fixed: "0.5", percentage: 0.001 },
    Polygon: { fixed: "0.5", percentage: 0.001 },
    Tron: { fixed: "1", percentage: 0.001 },
  },
};

export class FeeCalculationService {
  /**
   * Calculate withdrawal fee and received amount
   */
  static async calculate(
    asset: string,
    amount: string,
    chain?: string
  ): Promise<FeeCalculationResult> {
    const validated = feeCalculationSchema.parse({ asset, amount, chain });
    const selectedChain = chain || this.getDefaultChain(asset);

    logger.info(
      { asset: validated.asset, amount: validated.amount, chain: selectedChain },
      "Calculating withdrawal fees"
    );

    try {
      // Get asset ID for Safeheron
      const assetId = safeheronService.getAssetId(validated.asset, selectedChain);

      // Try to get fee from Safeheron API
      const safeheronFee = await safeheronService.estimateFee(
        assetId,
        validated.amount,
        "pending_address" // Placeholder for fee estimation
      );

      if (safeheronFee) {
        const feeAmount = safeheronFee.fee;
        const numericAmount = parseFloat(validated.amount);
        const numericFee = parseFloat(feeAmount);
        const receivedAmount = Math.max(0, numericAmount - numericFee).toFixed(7);

        return {
          asset: validated.asset,
          amount: validated.amount,
          chain: selectedChain,
          fee: feeAmount,
          receivedAmount,
          feePercentage: (numericFee / numericAmount) * 100,
        };
      }
    } catch (error: any) {
      logger.warn(
        { error: error.message, asset: validated.asset },
        "Safeheron fee estimation failed, using fallback"
      );
    }

    // Fallback to local fee calculation
    return this.calculateFallback(validated.asset, validated.amount, selectedChain);
  }

  /**
   * Fallback fee calculation using local rates
   */
  private static calculateFallback(
    asset: string,
    amount: string,
    chain: string
  ): FeeCalculationResult {
    const rates = FEE_RATES[asset]?.[chain];
    const defaultRates = FEE_RATES[asset]?.Ethereum || { fixed: "0.001", percentage: 0.001 };
    const selectedRates = rates || defaultRates;

    const numericAmount = parseFloat(amount);
    const fixedFee = parseFloat(selectedRates.fixed);
    const percentageFee = numericAmount * selectedRates.percentage;
    const totalFee = Math.max(fixedFee, percentageFee);
    const receivedAmount = Math.max(0, numericAmount - totalFee).toFixed(7);

    return {
      asset,
      amount,
      chain,
      fee: totalFee.toFixed(7),
      receivedAmount,
      feePercentage: (totalFee / numericAmount) * 100,
    };
  }

  /**
   * Get default chain for an asset
   */
  static getDefaultChain(asset: string): string {
    const defaults: Record<string, string> = {
      BTC: "Bitcoin",
      ETH: "Ethereum",
      USDC: "Ethereum",
      USDT: "Ethereum",
    };
    return defaults[asset] || "Ethereum";
  }

  /**
   * Validate amount against available balance
   */
  static async validateAmount(
    asset: string,
    amount: string,
    availableBalance: string
  ): Promise<{ valid: boolean; error?: string; maxAmount?: string }> {
    const numericAmount = parseFloat(amount);
    const numericBalance = parseFloat(availableBalance);

    if (isNaN(numericAmount) || numericAmount <= 0) {
      return { valid: false, error: "Invalid amount" };
    }

    if (numericAmount > numericBalance) {
      const fee = await this.calculateFeeOnly(asset, numericAmount);
       const maxAmount = Math.max(0, numericBalance - parseFloat(fee.fee)).toFixed(7);
      return {
        valid: false,
        error: `Insufficient balance. Maximum withdrawal: ${maxAmount} ${asset}`,
        maxAmount,
      };
    }

    // Check minimum amount
    const minAmounts: Record<string, number> = {
      BTC: 0.001,
      ETH: 0.01,
      USDC: 1,
      USDT: 1,
    };

    const minAmount = minAmounts[asset] || 0.01;
    if (numericAmount < minAmount) {
      return {
        valid: false,
        error: `Minimum withdrawal amount: ${minAmount} ${asset}`,
      };
    }

    return { valid: true };
  }

  /**
   * Calculate fee only (without received amount)
   */
  static async calculateFeeOnly(
    asset: string,
    amount: number
  ): Promise<{ fee: string; feePercentage: number }> {
    const result = await this.calculate(asset, amount.toString());
    return {
      fee: result.fee,
      feePercentage: result.feePercentage,
    };
  }
}

// Export singleton instance
export const feeCalculationService = FeeCalculationService;
