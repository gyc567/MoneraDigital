import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Copy, Check, AlertTriangle, Wallet } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import QRCode from "qrcode";
import { useNavigate } from "react-router-dom";
import { format } from "date-fns";

const Deposit = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [asset, setAsset] = useState("USDT");
  const [network, setNetwork] = useState("TRON");
  const [copied, setCopied] = useState(false);
  const [qrCodeUrl, setQrCodeUrl] = useState("");

  const { data: walletInfo, isLoading: isWalletLoading } = useQuery({
    queryKey: ["walletInfo"],
    queryFn: async () => {
      const res = await fetch("/api/wallet/info", {
        headers: { Authorization: `Bearer ${localStorage.getItem("token")}` },
      });
      if (!res.ok) throw new Error("Failed to fetch wallet info");
      return res.json();
    },
  });

  const { data: depositsData } = useQuery({
    queryKey: ["deposits"],
    queryFn: async () => {
      const res = await fetch("/api/deposits?limit=10", {
        headers: { Authorization: `Bearer ${localStorage.getItem("token")}` },
      });
      if (!res.ok) throw new Error("Failed to fetch deposits");
      return res.json();
    },
    refetchInterval: 15000, // Poll every 15s
  });

  // Parse addresses from JSON string if available
  // Handle sql.NullString format { String: "...", Valid: true }
  let address = "";
  try {
      if (walletInfo?.addresses && walletInfo.addresses.Valid) {
          const addrMap = JSON.parse(walletInfo.addresses.String);
          address = addrMap[network] || "";
      }
  } catch (e) {
      console.error("Failed to parse wallet addresses", e);
  }

  useEffect(() => {
    if (address) {
      QRCode.toDataURL(address)
        .then((url) => setQrCodeUrl(url))
        .catch((err) => console.error(err));
    } else {
        setQrCodeUrl("");
    }
  }, [address]);

  const copyToClipboard = async () => {
    if (address) {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      toast.success(t("deposit.copied"));
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (isWalletLoading) return <div className="p-8 text-center">Loading...</div>;

  // Check if wallet is created (SUCCESS)
  if (!walletInfo || walletInfo.status !== "SUCCESS") {
      return (
        <div className="flex flex-col items-center justify-center h-[60vh] space-y-4 animate-fade-in">
            <div className="p-4 bg-secondary rounded-full">
                <Wallet className="w-12 h-12 text-muted-foreground" />
            </div>
            <h2 className="text-xl font-semibold">{t("deposit.activate")}</h2>
            <p className="text-muted-foreground text-center max-w-md">{t("deposit.activateDesc")}</p>
            <Button onClick={() => navigate("/dashboard/account-opening")}>
                {t("deposit.activate")}
            </Button>
        </div>
      )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t("deposit.title")}</h1>
        <p className="text-muted-foreground mt-2">{t("deposit.description")}</p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Deposit Form */}
        <Card className="border-border/50 bg-card/50 backdrop-blur-sm">
          <CardHeader>
            <CardTitle>{t("deposit.address")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("deposit.selectAsset")}</label>
                <Select value={asset} onValueChange={setAsset}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="USDT">USDT</SelectItem>
                    <SelectItem value="USDC">USDC</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">{t("deposit.selectNetwork")}</label>
                <Select value={network} onValueChange={setNetwork}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="TRON">TRON (TRC20)</SelectItem>
                    <SelectItem value="ETH">Ethereum (ERC20)</SelectItem>
                    <SelectItem value="BSC">BNB Smart Chain (BEP20)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {address ? (
                <div className="flex flex-col items-center space-y-4 p-6 bg-secondary/30 rounded-lg border border-border/50">
                    {qrCodeUrl && <img src={qrCodeUrl} alt="QR Code" className="w-40 h-40 rounded-lg bg-white p-2" />}
                    
                    <div className="w-full space-y-2">
                        <div className="flex items-center gap-2 p-3 bg-background rounded border border-border font-mono text-sm break-all text-center justify-center">
                            <span>{address}</span>
                        </div>
                        <Button variant="outline" className="w-full" onClick={copyToClipboard}>
                            {copied ? <Check className="w-4 h-4 mr-2 text-green-500" /> : <Copy className="w-4 h-4 mr-2" />}
                            {t("deposit.copy")}
                        </Button>
                    </div>

                    <div className="flex items-start gap-2 text-sm text-yellow-600 dark:text-yellow-500 bg-yellow-500/10 p-3 rounded-md w-full">
                        <AlertTriangle className="w-5 h-5 shrink-0" />
                        <p>{t("deposit.warning", { network })}</p>
                    </div>
                    
                    <p className="text-xs text-muted-foreground">{t("deposit.minDeposit")}</p>
                </div>
            ) : (
                <div className="text-center py-8 text-muted-foreground bg-secondary/20 rounded-lg">
                    Address not available for this network.
                </div>
            )}
          </CardContent>
        </Card>

        {/* History */}
        <Card className="border-border/50 bg-card/50 backdrop-blur-sm">
            <CardHeader>
                <CardTitle>{t("deposit.history")}</CardTitle>
            </CardHeader>
            <CardContent>
                {!depositsData?.deposits || depositsData.deposits.length === 0 ? (
                    <div className="text-center py-12 text-muted-foreground">
                        {t("deposit.noHistory")}
                    </div>
                ) : (
                    <div className="space-y-4">
                        {depositsData.deposits.map((tx: any) => (
                            <div key={tx.id} className="flex items-center justify-between p-4 border border-border/50 bg-card rounded-lg hover:bg-secondary/20 transition-colors">
                                <div>
                                    <div className="font-medium flex items-center gap-2">
                                        {tx.asset} 
                                        <span className="text-xs px-2 py-0.5 rounded-full bg-secondary text-muted-foreground font-normal">{tx.chain}</span>
                                    </div>
                                    <div className="text-xs text-muted-foreground mt-1">
                                        {format(new Date(tx.created_at), "yyyy-MM-dd HH:mm")}
                                    </div>
                                </div>
                                <div className="text-right">
                                    <div className="font-bold text-green-600 dark:text-green-400">+{parseFloat(tx.amount).toFixed(2)}</div>
                                    <div className={`text-xs mt-1 font-medium ${
                                        tx.status === 'CONFIRMED' ? 'text-green-500' : 
                                        tx.status === 'PENDING' ? 'text-yellow-500' : 'text-red-500'
                                    }`}>
                                        {t(`deposit.status.${tx.status}`)}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default Deposit;
