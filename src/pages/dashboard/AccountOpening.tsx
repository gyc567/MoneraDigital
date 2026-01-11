import { useTranslation } from "react-i18next";
import { Shield, Copy, Check, AlertCircle, Loader2, Wallet } from "lucide-react";
import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";

const AccountOpening = () => {
  const { t } = useTranslation();
  const { toast } = useToast();
  const [status, setStatus] = useState<"none" | "loading" | "success" | "error">("none");
  const [walletData, setWalletData] = useState<{ address: string; walletId: string } | null>(null);
  const [copied, setCopied] = useState(false);

  // Simulated function to create wallet - replace with actual API call
  const handleCreateWallet = async () => {
    setStatus("loading");

    try {
      // Simulate API call to Safeheron
      await new Promise((resolve) => setTimeout(resolve, 2000));

      // Mock successful response
      setWalletData({
        address: "0x1234567890abcdef1234567890abcdef12345678",
        walletId: "wallet_sfh_abc123def456",
      });
      setStatus("success");

      toast({
        title: t("wallet.opening.successTitle"),
        description: t("wallet.opening.successDescription"),
      });
    } catch (error) {
      setStatus("error");
      toast({
        variant: "destructive",
        title: t("wallet.opening.errorTitle"),
        description: t("wallet.opening.errorDescription"),
      });
    }
  };

  const copyToClipboard = async () => {
    if (walletData?.address) {
      await navigator.clipboard.writeText(walletData.address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

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
          {status === "none" && (
            <Button
              onClick={handleCreateWallet}
              className="w-full h-12 text-base font-medium"
              size="lg"
            >
              <Wallet className="w-5 h-5 mr-2" />
              {t("wallet.opening.activateButton")}
            </Button>
          )}

          {/* Loading State */}
          {status === "loading" && (
            <div className="flex flex-col items-center justify-center py-8 space-y-4">
              <Loader2 className="w-12 h-12 text-primary animate-spin" />
              <p className="text-muted-foreground">{t("wallet.opening.creating")}</p>
            </div>
          )}

          {/* Success State */}
          {status === "success" && walletData && (
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
                  {t("wallet.opening.walletId")}: {walletData.walletId}
                </p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 text-sm font-mono break-all bg-background p-3 rounded-md">
                    {walletData.address}
                  </code>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={copyToClipboard}
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

          {/* Error State */}
          {status === "error" && (
            <div className="p-4 rounded-lg bg-destructive/10 border border-destructive/20">
              <div className="flex items-center gap-2 text-destructive mb-2">
                <AlertCircle className="w-5 h-5" />
                <span className="font-medium">{t("wallet.opening.error")}</span>
              </div>
              <Button
                onClick={handleCreateWallet}
                variant="outline"
                className="w-full"
              >
                {t("wallet.opening.retry")}
              </Button>
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
