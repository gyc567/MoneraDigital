import { useState, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ArrowRight, CheckCircle2, Clock, AlertCircle, Shield, Wallet, Plus } from "lucide-react";
import { toast } from "sonner";

interface WithdrawalAddress {
  id: number;
  wallet_address: string;
  chain_type: string;
  address_alias: string;
  verified: boolean;
}

interface Withdrawal {
  id: number;
  status: "PENDING" | "PROCESSING" | "SENT" | "CONFIRMING" | "CONFIRMED" | "COMPLETED" | "FAILED";
  amount: string;
  coin_type: string;
  to_address: string;
  transaction_hash: string | null;
  network_fee: string | null;
  actual_amount: string | null;
  created_at: string;
  completed_at: string | null;
}

interface ChainOption {
  value: string;
  label: string;
  feeEstimate: string;
}

const CHAIN_OPTIONS: Record<string, ChainOption[]> = {
  BTC: [
    { value: "Bitcoin", label: "Bitcoin Network", feeEstimate: "0.0005" },
  ],
  ETH: [
    { value: "Ethereum", label: "Ethereum Network", feeEstimate: "0.002" },
  ],
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

// Address validation patterns by chain type
const ADDRESS_PATTERNS: Record<string, RegExp> = {
  Ethereum: /^0x[a-fA-F0-9]{40}$/,
  Bitcoin: /^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^bc1[a-zA-HJ-NP-Z0-9]{25,90}$/,
  Arbitrum: /^0x[a-fA-F0-9]{40}$/,
  Polygon: /^0x[a-fA-F0-9]{40}$/,
  Tron: /^T[a-zA-Z0-9]{33}$/,
};

function isValidAddress(address: string, chain: string): boolean {
  const pattern = ADDRESS_PATTERNS[chain];
  return pattern ? pattern.test(address) : address.length >= 26 && address.length <= 64;
}

function Withdraw() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const [addresses, setAddresses] = useState<WithdrawalAddress[]>([]);
  const [withdrawalHistory, setWithdrawalHistory] = useState<Withdrawal[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isHistoryLoading, setIsHistoryLoading] = useState(false);
  // Address management state
  const [isAddAddressOpen, setIsAddAddressOpen] = useState(false);
  const [isCreatingAddress, setIsCreatingAddress] = useState(false);
  const [newAddressAlias, setNewAddressAlias] = useState("");
  const [newAddress, setNewAddress] = useState("");
  const [newAddressChain, setNewAddressChain] = useState("Ethereum");

  // Withdrawal form state
  const [selectedAddressId, setSelectedAddressId] = useState<string>("");
  const [amount, setAmount] = useState("");
  const initialAsset = (searchParams.get('asset') as "BTC" | "ETH" | "USDC" | "USDT") || "ETH";
  const [asset, setAsset] = useState<"BTC" | "ETH" | "USDC" | "USDT">(initialAsset);
  const [chain, setChain] = useState<string>("");
  const [fee, setFee] = useState<string>("0");
  const [receivedAmount, setReceivedAmount] = useState<string>("0");
  const [isCalculatingFee, setIsCalculatingFee] = useState(false);
  const [isConfirming, setIsConfirming] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedAddress, setSelectedAddress] = useState<WithdrawalAddress | null>(null);
  const [is2FADialogOpen, setIs2FADialogOpen] = useState(false);
  const [twoFactorCode, setTwoFactorCode] = useState("");
  const [isVerifying2FA, setIsVerifying2FA] = useState(false);
  const [pendingWithdrawal, setPendingWithdrawal] = useState<{
    addressId: number;
    amount: string;
    asset: string;
    chain: string;
  } | null>(null);

  // Fetch verified addresses
  const fetchAddresses = async () => {
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/addresses", {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        // Backend returns snake_case
        const allAddresses: WithdrawalAddress[] = data.addresses || [];
        // Only show verified addresses for withdrawal
        setAddresses(allAddresses.filter(a => a.verified));

        // Logic for auto-select
        // NOTE: Backend Address doesn't strictly have 'addressType' that matches asset EXACTLY like 'ETH' vs 'Ethereum'.
        // chain_type in backend is stored from req.ChainType.
        // We assume User selects Asset, then we filter addresses by ChainType?
        // But Asset != ChainType (USDT can be on TRC20).
        // UI logic: Select Asset -> Select Chain -> Select Address.
        // Current UI: Select Address -> it implies Chain & Asset.
        // OR Select Asset -> UI sets Chain Options -> User selects Chain.
        
        // Let's assume for now we just show all addresses.
        if (data.addresses && data.addresses.length > 0) {
             // Basic auto-select first
             const first = data.addresses[0];
             setSelectedAddressId(String(first.id));
             setSelectedAddress(first);
        }
      }
    } catch (error) {
      console.error("Fetch addresses error:", error);
      toast.error("Failed to fetch addresses");
    } finally {
      setIsLoading(false);
    }
  };

  // Create new address
  const handleCreateAddress = async () => {
    if (isCreatingAddress) return;

    if (!newAddressAlias || !newAddress || !newAddressChain) {
      toast.error("Please fill in all fields");
      return;
    }

    if (!isValidAddress(newAddress, newAddressChain)) {
      toast.error(`Invalid ${newAddressChain} address format. Please check and try again.`);
      return;
    }

    setIsCreatingAddress(true);
    try {
      const token = localStorage.getItem("token");
      const csrfToken = localStorage.getItem("csrf_token");

      const res = await fetch("/api/addresses", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
          ...(csrfToken && { "X-CSRF-Token": csrfToken }),
        },
        body: JSON.stringify({
          wallet_address: newAddress.trim(),
          chain_type: newAddressChain,
          address_alias: newAddressAlias.trim(),
        }),
      });

      const data = await res.json();
      if (res.ok) {
        toast.success(t("addresses.addSuccessMessage"));
        setIsAddAddressOpen(false);
        setNewAddressAlias("");
        setNewAddress("");
        setNewAddressChain("Ethereum");
        await fetchAddresses();
      } else {
        toast.error(data.error || "Failed to add address");
      }
    } catch (error) {
      console.error("Create address error:", error);
      toast.error("Failed to add address. Please try again.");
    } finally {
      setIsCreatingAddress(false);
    }
  };

  // Reset address form to initial state
  function resetAddressForm() {
    setIsAddAddressOpen(false);
    setNewAddressAlias("");
    setNewAddress("");
    setNewAddressChain("Ethereum");
  }

  // Fetch withdrawal history
  const fetchHistory = async () => {
    setIsHistoryLoading(true);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/withdrawals", {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        setWithdrawalHistory(data.withdrawals || []);
      }
    } catch (error) {
      console.error("Fetch history error:", error);
    } finally {
      setIsHistoryLoading(false);
    }
  };

  // Calculate fees when amount or chain changes
  const calculateFees = async () => {
    if (!amount || !chain || parseFloat(amount) <= 0) {
      setFee("0");
      setReceivedAmount("0");
      return;
    }

    setIsCalculatingFee(true);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`/api/withdrawals/fees?asset=${asset}&amount=${amount}&chain=${chain}`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        setFee(data.fee);
        setReceivedAmount(data.receivedAmount);
      } else {
        setFee("0");
        setReceivedAmount("0");
      }
    } catch (error) {
      console.error("Fee calculation error:", error);
      setFee("0");
      setReceivedAmount("0");
    } finally {
      setIsCalculatingFee(false);
    }
  };

  useEffect(() => {
    fetchAddresses();
    fetchHistory();
  }, []);

  useEffect(() => {
    // Set default chain based on asset
    const defaultChains: Record<string, string> = {
      BTC: "Bitcoin",
      ETH: "Ethereum",
      USDC: "Ethereum",
      USDT: "Ethereum",
    };
    setChain(defaultChains[asset] || "Ethereum");

    // Reset fee calculation when asset changes
    setFee("0");
    setReceivedAmount("0");
  }, [asset]);

  useEffect(() => {
    calculateFees();
  }, [amount, chain, asset]);

  const handleAddressChange = (addressId: string) => {
    setSelectedAddressId(addressId);
    const addr = addresses.find((a) => a.id === Number(addressId));
    if (addr) {
      setSelectedAddress(addr);
      // setAsset(addr.chain_type); // Mapping needed if enabled
    }
  };

  const handleSubmitWithdrawal = async () => {
    if (!selectedAddressId || !amount) {
      toast.error("Please fill in all fields");
      return;
    }

    setIsSubmitting(true);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/withdrawals", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        // Match Backend snake_case
        body: JSON.stringify({
          address_id: Number(selectedAddressId),
          amount,
          asset,
        }),
      });

      const data = await res.json();
      if (res.ok) {
        toast.success(data.message || "Withdrawal initiated successfully");
        setAmount("");
        setIsConfirming(false);
        await fetchHistory();
      } else {
        toast.error(data.error || "Failed to initiate withdrawal");
      }
    } catch (error) {
      console.error("Withdraw error:", error);
      toast.error("Failed to initiate withdrawal");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle 2FA verification and submit withdrawal
  const handleVerify2FAAndSubmit = async () => {
    if (!pendingWithdrawal || !twoFactorCode) {
      toast.error("Missing verification data");
      return;
    }

    setIsVerifying2FA(true);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/withdrawals/verify-2fa", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          ...pendingWithdrawal,
          twoFactorCode,
        }),
      });

      const data = await res.json();
      if (res.ok) {
        toast.success("Withdrawal initiated successfully");
        setAmount("");
        setTwoFactorCode("");
        setIs2FADialogOpen(false);
        setPendingWithdrawal(null);
        await fetchHistory();
      } else {
        toast.error(data.error || "Verification failed");
      }
    } catch (error) {
      console.error("2FA verification error:", error);
      toast.error("Verification failed");
    } finally {
      setIsVerifying2FA(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "COMPLETED":
        return <CheckCircle2 className="text-green-500" size={20} />;
      case "PENDING":
      case "PROCESSING":
      case "SENT":
      case "CONFIRMING":
      case "CONFIRMED":
        return <Clock className="text-yellow-500" size={20} />;
      case "FAILED":
        return <AlertCircle className="text-red-500" size={20} />;
      default:
        return null;
    }
  };

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">{t("dashboard.withdraw.title")}</h1>
      </div>

      <Tabs defaultValue="withdraw" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="withdraw">{t("dashboard.withdraw.tab.new")}</TabsTrigger>
          <TabsTrigger value="history">{t("dashboard.withdraw.tab.history")}</TabsTrigger>
        </TabsList>

        <TabsContent value="withdraw" className="space-y-6">
          {isLoading ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground">{t("dashboard.withdraw.address.loading")}</div>
              </CardContent>
            </Card>
          ) : addresses.length === 0 ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground mb-6">
                  {t("dashboard.withdraw.address.empty")}
                </div>
                <div className="flex justify-center">
                  <Button onClick={() => setIsAddAddressOpen(true)} className="gap-2">
                    <Plus size={16} />
                    {t("dashboard.withdraw.address.addButton")}
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : (
            <>
              <Card className="bg-card/50 border-border/50">
                <CardHeader>
                  <CardTitle className="text-lg">{t("dashboard.withdraw.address.title")}</CardTitle>
                  <CardDescription>{t("dashboard.withdraw.address.description")}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="address">{t("dashboard.withdraw.address.label")}</Label>
                    <Select value={selectedAddressId} onValueChange={handleAddressChange}>
                      <SelectTrigger id="address">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {addresses.map((addr) => (
                          <SelectItem key={addr.id} value={String(addr.id)}>
                            <div className="flex items-center gap-2">
                              <span>{addr.address_alias}</span>
                              <Badge variant="outline">{addr.chain_type}</Badge>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {/* Chain Selection */}
                  <div className="space-y-2">
                    <Label htmlFor="chain">{t("dashboard.withdraw.chain.label")}</Label>
                    <Select value={chain} onValueChange={setChain}>
                      <SelectTrigger id="chain">
                        <SelectValue placeholder="Select chain" />
                      </SelectTrigger>
                      <SelectContent>
                        {(CHAIN_OPTIONS[asset] || []).map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            <div className="flex items-center justify-between gap-4">
                              <span>{option.label}</span>
                              <span className="text-xs text-muted-foreground">
                                ~{option.feeEstimate} {asset} fee
                              </span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {selectedAddress && (
                    <div className="p-4 bg-secondary/30 rounded-lg border border-border/50">
                      <p className="text-xs text-muted-foreground mb-2">{t("dashboard.withdraw.address.details")}</p>
                      <p className="text-sm font-mono break-all">{selectedAddress.wallet_address}</p>
                    </div>
                  )}
                </CardContent>
              </Card>

              <Card className="bg-card/50 border-border/50">
                <CardHeader>
                  <CardTitle className="text-lg">{t("dashboard.withdraw.amount.title")}</CardTitle>
                  <CardDescription>{t("dashboard.withdraw.amount.description")}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="amount">{t("dashboard.withdraw.amount.label")}</Label>
                    <div className="flex gap-2">
                      <Input
                        id="amount"
                        type="number"
                        placeholder="10.50"
                        value={amount}
                        onChange={(e) => setAmount(e.target.value)}
                        className="flex-1"
                      />
                      <div className="flex items-center px-4 rounded-md bg-secondary/30 border border-border">
                        <span className="font-semibold">{asset}</span>
                      </div>
                    </div>
                  </div>

                  {/* Fee and Received Amount Display */}
                  {amount && Number(amount) > 0 && (
                    <div className="p-4 bg-secondary/30 rounded-lg border border-border/50 space-y-3">
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">{t("dashboard.withdraw.fee.label")}</span>
                        <span className="font-medium">{fee} {asset}</span>
                      </div>
                      <div className="h-px bg-border/50" />
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">{t("dashboard.withdraw.receive.label")}</span>
                        <span className="font-semibold text-lg text-green-500">
                          {receivedAmount} {asset}
                        </span>
                      </div>
                    </div>
                  )}

                  <Button
                    onClick={() => setIsConfirming(true)}
                    disabled={!amount || !chain || Number(amount) <= 0 || isCalculatingFee}
                    className="w-full gap-2"
                  >
                    {isCalculatingFee ? (
                      <>{t("dashboard.withdraw.calculating")}</>
                    ) : (
                      <>
                        <ArrowRight size={16} />
                        {t("dashboard.withdraw.review.button")}
                      </>
                    )}
                  </Button>
                </CardContent>
              </Card>
            </>
          )}
        </TabsContent>

        <TabsContent value="history" className="space-y-4">
          {isHistoryLoading ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground">{t("dashboard.withdraw.history.loading")}</div>
              </CardContent>
            </Card>
          ) : withdrawalHistory.length === 0 ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground">{t("dashboard.withdraw.history.empty")}</div>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4">
              {withdrawalHistory.map((withdrawal) => (
                <Card key={withdrawal.id} className="bg-card/50 border-border/50">
                  <CardContent className="pt-6">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-3 flex-1">
                        <div className="mt-1">{getStatusIcon(withdrawal.status)}</div>
                        <div className="space-y-2 flex-1">
                           <div className="flex items-center gap-2">
                             <span className="font-semibold">{withdrawal.amount} {withdrawal.coin_type}</span>
                             <Badge variant={
                               withdrawal.status === "COMPLETED"
                                 ? "default"
                                 : withdrawal.status === "FAILED"
                                 ? "destructive"
                                 : "secondary"
                             }>
                               {withdrawal.status}
                             </Badge>
                           </div>
                           <p className="text-xs text-muted-foreground font-mono break-all">
                             To: {withdrawal.to_address}
                           </p>
                           <p className="text-xs text-muted-foreground">
                             {new Date(withdrawal.created_at).toLocaleString()}
                           </p>
                           <div className="flex gap-4 mt-2">
                             {withdrawal.network_fee && (
                               <p className="text-xs text-muted-foreground">
                                 Fee: {withdrawal.network_fee} {withdrawal.coin_type}
                               </p>
                             )}
                             {withdrawal.transaction_hash && (
                               <p className="text-xs text-muted-foreground font-mono break-all">
                                 Tx: {withdrawal.transaction_hash}
                               </p>
                             )}
                           </div>
                         </div>
                       </div>
                     </div>
                   </CardContent>
                 </Card>
               ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      {/* Confirmation Dialog */}
      <AlertDialog open={isConfirming} onOpenChange={setIsConfirming}>
        <AlertDialogContent className="sm:max-w-[425px]">
          <AlertDialogHeader>
            <AlertDialogTitle>{t("dashboard.withdraw.confirm.title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("dashboard.withdraw.confirm.description")}
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-muted-foreground">{t("dashboard.withdraw.amount.label")}</p>
                <p className="text-lg font-semibold">{amount} {asset}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t("dashboard.withdraw.fee.label")}</p>
                <p className="text-lg font-semibold">{fee} {asset}</p>
              </div>
            </div>

            <div className="p-3 rounded-lg bg-green-500/10 border border-green-500/20">
              <p className="text-xs text-green-200/80">
                {t("dashboard.withdraw.receive.label")}: <strong>{receivedAmount} {asset}</strong>
              </p>
            </div>

            <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
              <p className="text-xs text-blue-200/80">
                {t("dashboard.withdraw.confirm.to")}: <span className="font-mono break-all">{selectedAddress?.wallet_address}</span>
              </p>
              <p className="text-xs text-blue-200/80 mt-1">
                Chain: <strong>{chain}</strong>
              </p>
            </div>
          </div>

           <AlertDialogFooter>
             <AlertDialogCancel disabled={isSubmitting}>{t("common.cancel")}</AlertDialogCancel>
             <AlertDialogAction
               onClick={async () => {
                 // Check if this is a new address (first withdrawal to this address)
                 if (selectedAddress && !selectedAddress.verified) {
                   setIsConfirming(false);
                   setIs2FADialogOpen(true);
                   setPendingWithdrawal({
                     addressId: selectedAddress.id,
                     amount,
                     asset,
                     chain,
                   });
                 } else {
                   await handleSubmitWithdrawal();
                 }
               }}
               disabled={isSubmitting}
               className="bg-green-600 hover:bg-green-700"
             >
               {isSubmitting ? t("dashboard.withdraw.processing") : t("dashboard.withdraw.confirm.button")}
             </AlertDialogAction>
           </AlertDialogFooter>
         </AlertDialogContent>
       </AlertDialog>

       {/* 2FA Verification Dialog for New Addresses */}
       <Dialog open={is2FADialogOpen} onOpenChange={setIs2FADialogOpen}>
         <DialogContent className="sm:max-w-[425px]">
           <DialogHeader>
             <DialogTitle className="flex items-center gap-2">
               <Shield className="h-5 w-5 text-yellow-500" />
               {t("auth.2fa.required")}
             </DialogTitle>
             <DialogDescription>
               {t("auth.2fa.firstWithdrawal")}
             </DialogDescription>
           </DialogHeader>

           <div className="space-y-4 py-4">
             <div className="space-y-2">
               <Label htmlFor="2fa-code">{t("auth.2fa.code")}</Label>
               <Input
                 id="2fa-code"
                 placeholder={t("auth.2fa.placeholder")}
                 value={twoFactorCode}
                 onChange={(e) => setTwoFactorCode(e.target.value)}
                 maxLength={6}
               />
               <p className="text-xs text-muted-foreground">
                {t("auth.2fa.hint")}
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIs2FADialogOpen(false);
                setTwoFactorCode("");
                setPendingWithdrawal(null);
              }}
              disabled={isVerifying2FA}
            >
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handleVerify2FAAndSubmit}
              disabled={isVerifying2FA || !twoFactorCode}
            >
              {isVerifying2FA ? t("auth.2fa.verifying") : t("auth.2fa.verifyButton")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add Address Dialog */}
      <Dialog open={isAddAddressOpen} onOpenChange={setIsAddAddressOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Plus className="h-5 w-5" />
              {t("addresses.addDialog")}
            </DialogTitle>
            <DialogDescription>
              {t("addresses.addDescription")}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="alias">{t("addresses.label")}</Label>
              <Input
                id="alias"
                placeholder={t("addresses.placeholder")}
                value={newAddressAlias}
                onChange={(e) => setNewAddressAlias(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                {t("addresses.hint")}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="chain">{t("deposit.selectNetwork")}</Label>
              <Select value={newAddressChain} onValueChange={setNewAddressChain}>
                <SelectTrigger id="chain">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Ethereum">Ethereum</SelectItem>
                  <SelectItem value="Arbitrum">Arbitrum</SelectItem>
                  <SelectItem value="Polygon">Polygon</SelectItem>
                  <SelectItem value="Tron">Tron</SelectItem>
                  <SelectItem value="Bitcoin">Bitcoin</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="walletAddress">{t("deposit.address")}</Label>
              <Input
                id="walletAddress"
                placeholder={t("addresses.addressPlaceholder")}
                value={newAddress}
                onChange={(e) => setNewAddress(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                {t("addresses.addressHint")}
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={resetAddressForm}
              disabled={isCreatingAddress}
            >
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handleCreateAddress}
              disabled={isCreatingAddress || !newAddressAlias || !newAddress}
            >
              {isCreatingAddress ? t("addresses.adding") : t("addresses.addButton")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Withdraw;