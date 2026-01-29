import { useTranslation } from "react-i18next";
import { Shield, Copy, Check, AlertCircle, Loader2, Wallet, Globe } from "lucide-react";
import { useState, useEffect, useMemo } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiRequest } from "@/lib/api-client";

// SQL NullString format from backend (sql.NullString serializes to {String, Valid})
interface NullString {
  String: string;
  Valid: boolean;
}

// Wallet info response type - matches backend WalletCreationRequest structure
interface WalletInfoResponse {
  status: string;
  walletId?: NullString;
  address?: NullString;
  addresses?: NullString | string; // Can be NullString or plain string
}

// Network option type
interface NetworkOption {
  value: string;
  label: string;
}

// Parse addresses JSON and extract available networks
export const parseAvailableNetworks = (addressesJson: string): NetworkOption[] => {
  if (!addressesJson) return [];
  try {
    const addrMap = JSON.parse(addressesJson);
    return Object.keys(addrMap).map((network) => ({
      value: network,
      label: network,
    }));
  } catch {
    return [];
  }
};

// Get display address for selected network
export const getDisplayAddress = (addressesJson: string, selectedNetwork: string): string => {
  if (!addressesJson) return "";
  try {
    const addrMap = JSON.parse(addressesJson);
    return addrMap[selectedNetwork] || "";
  } catch {
    return "";
  }
};

// Currency options
const CURRENCY_OPTIONS = [
  { value: "USDT_ERC20", label: "USDT (ERC20)" },
  { value: "USDT_TRC20", label: "USDT (TRC20)" },
  { value: "USDT_BSC", label: "USDT (BSC)" },
  { value: "ETH", label: "Ethereum (ETH)" },
  { value: "TRON", label: "TRON (TRX)" },
  { value: "BSC", label: "BSC (BNB)" },
];

const AccountOpening = () => {
  const { t } = useTranslation();
  const { toast } = useToast();
  const navigate = useNavigate();
  const location = useLocation();
  const [copied, setCopied] = useState(false);
  const [selectedCurrency, setSelectedCurrency] = useState("TRON");
  const queryClient = useQueryClient();

  // Get token for API calls
  const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;

  // Auth guard: redirect to login if not authenticated
  useEffect(() => {
    if (!token) {
      toast({
        variant: "destructive",
        title: t("auth.required"),
        description: t("auth.pleaseLoginFirst"),
      });
      navigate("/login", { state: { returnTo: location.pathname } });
    }
  }, [token, navigate, location.pathname, toast, t]);

  const { data: walletInfo, isLoading } = useQuery<WalletInfoResponse>({
    queryKey: ["walletInfo"],
    enabled: !!token,
    queryFn: async () => {
      if (!token) {
        return { status: "NONE" };
      }
      return apiRequest("/api/wallet/info", {
        headers: { Authorization: `Bearer ${token}` },
      });
    },
    refetchInterval: (query) => {
      const data = query.state.data;
      return data?.status === "CREATING" ? 2000 : false;
    },
  });

  // Parse addresses and extract available networks
  const addressesJson = useMemo(() => {
    if (!walletInfo?.addresses) return "";
    return typeof walletInfo.addresses === "string"
      ? walletInfo.addresses
      : walletInfo.addresses.String || "";
  }, [walletInfo?.addresses]);

  const availableNetworks = useMemo(
    () => parseAvailableNetworks(addressesJson),
    [addressesJson]
  );

  const [selectedNetwork, setSelectedNetwork] = useState<string>("");

  useEffect(() => {
    if (availableNetworks.length > 0 && !selectedNetwork) {
      setSelectedNetwork(availableNetworks[0].value);
    }
  }, [availableNetworks, selectedNetwork]);

  const createMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        throw new Error("Not authenticated");
      }
      return apiRequest("/api/wallet/create", {
        method: "POST",
        headers: {
            Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
          productCode: "X_FINANCE",
          currency: selectedCurrency
        })
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["walletInfo"] });
    },
    onError: (err) => {
      toast({
        variant: "destructive",
        title: t("wallet.opening.errorTitle"),
        description: err.message,
      });
    }
  });

  const handleCreateWallet = () => {
    createMutation.mutate();
  };

  const copyToClipboard = async (text: string) => {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
  };

  const status = walletInfo?.status || "NONE";
  const isCreating = status === "CREATING" || createMutation.isPending;

  const displayAddress = useMemo(
    () => getDisplayAddress(addressesJson, selectedNetwork),
    [addressesJson, selectedNetwork]
  );

  const walletId = walletInfo?.walletId?.String || walletInfo?.address?.String || "";

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t("wallet.opening.title")}</h1>
        <p className="text-muted-foreground mt-2">
          {t("wallet.opening.description")}
        </p>
      </div>

      {/* Main Card */}
      <Card className="bg-card/50 backdrop-blur-sm border-border/50">
        <CardHeader className="text-center pb-2">
          <div className="mx-auto mb-4 p-4 rounded-full bg-primary/10">
            <Shield className="w-12 h-12 text-primary" />
          </div>
          <CardTitle className="text-xl">{t("wallet.opening.cardTitle")}</CardTitle>
          <CardDescription className="max-w-md mx-auto">
            {t("wallet.opening.cardDescription")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Security Info */}
          <div className="flex items-center gap-3 p-4 rounded-lg bg-secondary/50 text-sm">
            <AlertCircle className="w-5 h-5 text-primary shrink-0" />
            <p>{t("wallet.opening.securityInfo")}</p>
          </div>

          {/* Currency Selection */}
          {(status === "NONE" || status === "FAILED") && (
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("wallet.opening.currencyLabel") || "Select Currency"}</label>
              <Select value={selectedCurrency} onValueChange={setSelectedCurrency}>
                <SelectTrigger>
                  <SelectValue placeholder="Select currency" />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCY_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Action Button */}
          {(status === "NONE" || status === "FAILED") && (
            <Button
              onClick={handleCreateWallet}
              className="w-full h-12 text-base font-medium"
              size="lg"
              disabled={isCreating}
            >
              {isCreating ? (
                  <Loader2 className="w-5 h-5 mr-2 animate-spin" />
              ) : (
                  <Wallet className="w-5 h-5 mr-2" />
              )}
              {isCreating ? t("wallet.opening.creating") : t("wallet.opening.activateButton")}
            </Button>
          )}

          {/* Loading State Display */}
          {status === "CREATING" && !createMutation.isPending && (
             <div className="flex flex-col items-center justify-center py-8 space-y-4">
               <Loader2 className="w-12 h-12 text-primary animate-spin" />
               <p className="text-muted-foreground">{t("wallet.opening.creating")}</p>
             </div>
          )}

              {/* Success State */}
          {status === "SUCCESS" && (
            <div className="space-y-4">
              <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/20">
                <div className="flex items-center gap-2 text-green-500 mb-2">
                  <Check className="w-5 h-5" />
                  <span className="font-medium">{t("wallet.opening.activated")}</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  {t("wallet.opening.addressLabel")}
                </p>
              </div>

              {/* Network Display */}
              {availableNetworks.length > 0 && (
                <>
                  {availableNetworks.length > 1 ? (
                    <Tabs value={selectedNetwork} onValueChange={setSelectedNetwork}>
                      <TabsList
                        className="grid w-full"
                        style={{
                          gridTemplateColumns: `repeat(${availableNetworks.length}, minmax(0, 1fr))`,
                        }}
                      >
                        {availableNetworks.map((network) => (
                          <TabsTrigger
                            key={network.value}
                            value={network.value}
                            className="flex items-center gap-1"
                          >
                            <Globe className="w-3 h-3" />
                            {network.label}
                          </TabsTrigger>
                        ))}
                      </TabsList>
                    </Tabs>
                  ) : (
                    <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-secondary/50 border border-border">
                      <Globe className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm font-medium">
                        {availableNetworks[0].label}
                      </span>
                    </div>
                  )}
                </>
              )}

              {/* Address Display */}
              <div className="p-4 rounded-lg bg-secondary/50 border border-border">
                <p className="text-xs text-muted-foreground mb-2">
                  {t("wallet.opening.walletId")}: {walletId}
                </p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 text-sm font-mono break-all bg-background p-3 rounded-md">
                    {displayAddress}
                  </code>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => copyToClipboard(displayAddress)}
                    className="shrink-0"
                  >
                    {copied ? (
                      <Check className="w-4 h-4 text-green-500" />
                    ) : (
                      <Copy className="w-4 h-4" />
                    )}
                  </Button>
                </div>
              </div>

              <p className="text-xs text-muted-foreground text-center">
                {t("wallet.opening.depositHint")}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Info Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="bg-card/30">
          <CardContent className="pt-6">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
              <Shield className="w-5 h-5 text-primary" />
            </div>
            <h3 className="font-medium mb-2">{t("wallet.opening.feature1Title")}</h3>
            <p className="text-sm text-muted-foreground">
              {t("wallet.opening.feature1Description")}
            </p>
          </CardContent>
        </Card>

        <Card className="bg-card/30">
          <CardContent className="pt-6">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
              <Wallet className="w-5 h-5 text-primary" />
            </div>
            <h3 className="font-medium mb-2">{t("wallet.opening.feature2Title")}</h3>
            <p className="text-sm text-muted-foreground">
              {t("wallet.opening.feature2Description")}
            </p>
          </CardContent>
        </Card>

        <Card className="bg-card/30">
          <CardContent className="pt-6">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
              <Check className="w-5 h-5 text-primary" />
            </div>
            <h3 className="font-medium mb-2">{t("wallet.opening.feature3Title")}</h3>
            <p className="text-sm text-muted-foreground">
              {t("wallet.opening.feature3Description")}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default AccountOpening;
