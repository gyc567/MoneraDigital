import { useTranslation } from "react-i18next";
import { Shield, Copy, Check, AlertCircle, Loader2, Wallet } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

const AccountOpening = () => {
  const { t } = useTranslation();
  const { toast } = useToast();
  const [copied, setCopied] = useState(false);
  const queryClient = useQueryClient();

  const { data: walletInfo, isLoading } = useQuery({
    queryKey: ["walletInfo"],
    queryFn: async () => {
      const res = await fetch("/api/wallet/info", {
        headers: { Authorization: `Bearer ${localStorage.getItem("token")}` },
      });
      if (!res.ok) throw new Error("Failed to fetch info");
      return res.json();
    },
    refetchInterval: (query) => {
        const data = query.state.data;
        return data?.status === "CREATING" ? 2000 : false
    }, 
  });

  const createMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch("/api/wallet/create", {
        method: "POST",
        headers: { 
            "Content-Type": "application/json",
            Authorization: `Bearer ${localStorage.getItem("token")}` 
        },
      });
      if (!res.ok) {
          const data = await res.json();
          throw new Error(data.error || "Failed to create wallet");
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["walletInfo"] });
      // Toast handled by UI state change usually, but good to notify
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

  // Parse address for display (Pick TRON or first available)
  let displayAddress = "";
  let walletId = "";
  if (status === "SUCCESS" && walletInfo?.addresses?.Valid) {
      try {
          const addrMap = JSON.parse(walletInfo.addresses.String);
          displayAddress = addrMap["TRON"] || addrMap["ETH"] || Object.values(addrMap)[0] as string || "";
          walletId = walletInfo.wallet_id?.String || "";
      } catch (e) {
          console.error("Failed to parse address", e);
      }
  }

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