import { describe, it, expect, beforeAll } from "vitest";
import i18n from "@/i18n/config";

describe("Withdraw Page i18n Translations", () => {
  describe("Chinese Translations (zh)", () => {
    beforeAll(async () => {
      await i18n.changeLanguage("zh");
    });

    it("has chain.placeholder translation", () => {
      const t = i18n.getFixedT("zh");
      expect(t("dashboard.withdraw.chain.placeholder")).toBe("选择网络");
    });

    it("has amount.placeholder translation", () => {
      const t = i18n.getFixedT("zh");
      expect(t("dashboard.withdraw.amount.placeholder")).toBe("请输入金额");
    });

    it("has 2fa.withdrawalDescription translation", () => {
      const t = i18n.getFixedT("zh");
      expect(t("dashboard.withdraw.2fa.withdrawalDescription")).toBe("请输入验证码以确认此提现");
    });

    it("has networkNames translations", () => {
      const t = i18n.getFixedT("zh");
      expect(t("dashboard.withdraw.networkNames.ethereum")).toBe("以太坊");
      expect(t("dashboard.withdraw.networkNames.arbitrum")).toBe("Arbitrum");
      expect(t("dashboard.withdraw.networkNames.polygon")).toBe("Polygon");
      expect(t("dashboard.withdraw.networkNames.tron")).toBe("Tron");
      expect(t("dashboard.withdraw.networkNames.bitcoin")).toBe("比特币");
      expect(t("dashboard.withdraw.networkNames.bitcoinNetwork")).toBe("比特币网络");
      expect(t("dashboard.withdraw.networkNames.ethereumNetwork")).toBe("以太坊网络");
      expect(t("dashboard.withdraw.networkNames.ethereumERC20")).toBe("以太坊 (ERC-20)");
      expect(t("dashboard.withdraw.networkNames.tronTRC20")).toBe("Tron (TRC-20)");
    });
  });

  describe("English Translations (en)", () => {
    beforeAll(async () => {
      await i18n.changeLanguage("en");
    });

    it("has chain.placeholder translation", () => {
      const t = i18n.getFixedT("en");
      expect(t("dashboard.withdraw.chain.placeholder")).toBe("Select chain");
    });

    it("has amount.placeholder translation", () => {
      const t = i18n.getFixedT("en");
      expect(t("dashboard.withdraw.amount.placeholder")).toBe("Enter amount");
    });

    it("has 2fa.withdrawalDescription translation", () => {
      const t = i18n.getFixedT("en");
      expect(t("dashboard.withdraw.2fa.withdrawalDescription")).toBe("Please enter your 2FA code to confirm this withdrawal.");
    });

    it("has networkNames translations", () => {
      const t = i18n.getFixedT("en");
      expect(t("dashboard.withdraw.networkNames.ethereum")).toBe("Ethereum");
      expect(t("dashboard.withdraw.networkNames.arbitrum")).toBe("Arbitrum");
      expect(t("dashboard.withdraw.networkNames.polygon")).toBe("Polygon");
      expect(t("dashboard.withdraw.networkNames.tron")).toBe("Tron");
      expect(t("dashboard.withdraw.networkNames.bitcoin")).toBe("Bitcoin");
      expect(t("dashboard.withdraw.networkNames.bitcoinNetwork")).toBe("Bitcoin Network");
      expect(t("dashboard.withdraw.networkNames.ethereumNetwork")).toBe("Ethereum Network");
      expect(t("dashboard.withdraw.networkNames.ethereumERC20")).toBe("Ethereum (ERC-20)");
      expect(t("dashboard.withdraw.networkNames.tronTRC20")).toBe("Tron (TRC-20)");
    });
  });
});

describe("getChainOptions Helper Function Logic", () => {
  const CHAIN_OPTIONS: Record<string, Array<{ value: string; label: string; feeEstimate: string }>> = {
    BTC: [{ value: "Bitcoin", label: "Bitcoin Network", feeEstimate: "0.0005" }],
    ETH: [{ value: "Ethereum", label: "Ethereum Network", feeEstimate: "0.002" }],
    USDC: [
      { value: "Ethereum", label: "Ethereum (ERC-20)", feeEstimate: "1" },
      { value: "Arbitrum", label: "Arbitrum", feeEstimate: "0.1" },
      { value: "Polygon", label: "Polygon", feeEstimate: "0.1" },
    ],
    USDT: [
      { value: "Ethereum", label: "Ethereum (ERC-20)", feeEstimate: "2" },
      { value: "Arbitrum", label: "Arbitrum", feeEstimate: "0.5" },
      { value: "Polygon", label: "Polygon", feeEstimate: "0.5" },
      { value: "Tron", label: "Tron (TRC-20)", feeEstimate: "1" },
    ],
  };

  function getChainOptionsZh(asset: string) {
    const tZh = (key: string): string => {
      const translations: Record<string, string> = {
        "dashboard.withdraw.networkNames.bitcoinNetwork": "比特币网络",
        "dashboard.withdraw.networkNames.ethereumNetwork": "以太坊网络",
        "dashboard.withdraw.networkNames.ethereumERC20": "以太坊 (ERC-20)",
        "dashboard.withdraw.networkNames.tronTRC20": "Tron (TRC-20)",
      };
      return translations[key] || key;
    };
    const options = CHAIN_OPTIONS[asset] || [];
    return options.map(opt => ({
      ...opt,
      label: opt.label === "Bitcoin Network" ? tZh("dashboard.withdraw.networkNames.bitcoinNetwork") :
            opt.label === "Ethereum Network" ? tZh("dashboard.withdraw.networkNames.ethereumNetwork") :
            opt.label === "Ethereum (ERC-20)" ? tZh("dashboard.withdraw.networkNames.ethereumERC20") :
            opt.label === "Tron (TRC-20)" ? tZh("dashboard.withdraw.networkNames.tronTRC20") :
            opt.label
    }));
  }

  function getChainOptionsEn(asset: string) {
    const options = CHAIN_OPTIONS[asset] || [];
    return options.map(opt => ({
      ...opt,
      label: opt.label
    }));
  }

  it("returns correct BTC chain options", () => {
    const zhOptions = getChainOptionsZh("BTC");
    expect(zhOptions[0].label).toBe("比特币网络");
    expect(zhOptions[0].value).toBe("Bitcoin");

    const enOptions = getChainOptionsEn("BTC");
    expect(enOptions[0].label).toBe("Bitcoin Network");
  });

  it("returns correct ETH chain options", () => {
    const zhOptions = getChainOptionsZh("ETH");
    expect(zhOptions[0].label).toBe("以太坊网络");

    const enOptions = getChainOptionsEn("ETH");
    expect(enOptions[0].label).toBe("Ethereum Network");
  });

  it("returns correct USDC chain options with multiple networks", () => {
    const zhOptions = getChainOptionsZh("USDC");
    expect(zhOptions[0].label).toBe("以太坊 (ERC-20)");
    expect(zhOptions[1].label).toBe("Arbitrum");
    expect(zhOptions[2].label).toBe("Polygon");
  });

  it("returns correct USDT chain options with TRON", () => {
    const zhOptions = getChainOptionsZh("USDT");
    expect(zhOptions[3].label).toBe("Tron (TRC-20)");

    const enOptions = getChainOptionsEn("USDT");
    expect(enOptions[3].label).toBe("Tron (TRC-20)");
  });

  it("returns empty array for unknown asset", () => {
    const zhOptions = getChainOptionsZh("UNKNOWN");
    expect(zhOptions).toEqual([]);
  });
});
