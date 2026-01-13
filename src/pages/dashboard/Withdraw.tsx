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
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ArrowRight, CheckCircle2, Clock, AlertCircle, Shield, Wallet } from "lucide-react";
import { toast } from "sonner";

interface WithdrawalAddress {
  id: number;
  address: string;
  addressType: "BTC" | "ETH" | "USDC" | "USDT";
  label: string;
  isVerified: boolean;
  isPrimary: boolean;
}

interface Withdrawal {
  id: number;
  status: "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED";
  amount: string;
  asset: string;
  toAddress: string;
  txHash: string | null;
  fee: string | null;
  receivedAmount: string | null;
  createdAt: string;
  completedAt: string | null;
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
    { value: "Ethereum", label: "Ethereum (ERC-20)", feeEstimate: "0.002" },
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

const Withdraw = () => {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const [addresses, setAddresses] = useState<WithdrawalAddress[]>([]);
  const [withdrawalHistory, setWithdrawalHistory] = useState<Withdrawal[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isHistoryLoading, setIsHistoryLoading] = useState(false);

  // Form states
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
        const verified = data.addresses.filter(
          (addr: WithdrawalAddress) => addr.isVerified && !addr.deactivatedAt
        );
        setAddresses(verified);

        // Auto-select address based on asset type preference
        const assetMatching = verified.filter((addr: WithdrawalAddress) => addr.addressType === asset);
        const primary = verified.find((addr: WithdrawalAddress) => addr.isPrimary);
        const primaryAssetMatch = assetMatching.find((addr: WithdrawalAddress) => addr.isPrimary);

        if (primaryAssetMatch) {
          setSelectedAddressId(String(primaryAssetMatch.id));
          setSelectedAddress(primaryAssetMatch);
        } else if (assetMatching.length > 0) {
          setSelectedAddressId(String(assetMatching[0].id));
          setSelectedAddress(assetMatching[0]);
        } else if (primary) {
          setSelectedAddressId(String(primary.id));
          setSelectedAddress(primary);
        } else if (verified.length > 0) {
          setSelectedAddressId(String(verified[0].id));
          setSelectedAddress(verified[0]);
        }
      }
    } catch (error) {
      console.error("Fetch addresses error:", error);
      toast.error("Failed to fetch addresses");
    } finally {
      setIsLoading(false);
    }
  };

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
      setAsset(addr.addressType);
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
        body: JSON.stringify({
          addressId: Number(selectedAddressId),
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
        <h1 className="text-3xl font-bold tracking-tight">Withdraw Funds</h1>
      </div>

      <Tabs defaultValue="withdraw" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="withdraw">New Withdrawal</TabsTrigger>
          <TabsTrigger value="history">History</TabsTrigger>
        </TabsList>

        <TabsContent value="withdraw" className="space-y-6">
          {isLoading ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground">Loading addresses...</div>
              </CardContent>
            </Card>
          ) : addresses.length === 0 ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground">
                  No verified addresses available. Please add and verify an address first.
                </div>
              </CardContent>
            </Card>
          ) : (
            <>
              <Card className="bg-card/50 border-border/50">
                <CardHeader>
                  <CardTitle className="text-lg">Select Withdrawal Address</CardTitle>
                  <CardDescription>Choose a verified address to withdraw funds to</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="address">Withdrawal Address</Label>
                    <Select value={selectedAddressId} onValueChange={handleAddressChange}>
                      <SelectTrigger id="address">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {addresses.map((addr) => (
                          <SelectItem key={addr.id} value={String(addr.id)}>
                            <div className="flex items-center gap-2">
                              <span>{addr.label}</span>
                              <Badge variant="outline">{addr.addressType}</Badge>
                              {addr.isPrimary && <Badge className="bg-blue-600">Primary</Badge>}
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {/* Chain Selection */}
                  <div className="space-y-2">
                    <Label htmlFor="chain">Withdrawal Chain</Label>
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
                      <p className="text-xs text-muted-foreground mb-2">Address Details</p>
                      <p className="text-sm font-mono break-all">{selectedAddress.address}</p>
                    </div>
                  )}
                </CardContent>
              </Card>

              <Card className="bg-card/50 border-border/50">
                <CardHeader>
                  <CardTitle className="text-lg">Withdrawal Amount</CardTitle>
                  <CardDescription>Enter the amount you wish to withdraw</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="amount">Amount</Label>
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
                        <span className="text-muted-foreground">Network Fee (estimated)</span>
                        <span className="font-medium">{fee} {asset}</span>
                      </div>
                      <div className="h-px bg-border/50" />
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">You will receive</span>
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
                      <>Calculating...</>
                    ) : (
                      <>
                        <ArrowRight size={16} />
                        Review & Confirm
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
                <div className="text-center text-muted-foreground">Loading history...</div>
              </CardContent>
            </Card>
          ) : withdrawalHistory.length === 0 ? (
            <Card className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="text-center text-muted-foreground">No withdrawal history yet</div>
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
                             <span className="font-semibold">{withdrawal.amount} {withdrawal.asset}</span>
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
                             To: {withdrawal.toAddress}
                           </p>
                           <p className="text-xs text-muted-foreground">
                             {new Date(withdrawal.createdAt).toLocaleString()}
                           </p>
                           <div className="flex gap-4 mt-2">
                             {withdrawal.fee && (
                               <p className="text-xs text-muted-foreground">
                                 Fee: {withdrawal.fee} {withdrawal.asset}
                               </p>
                             )}
                             {withdrawal.txHash && (
                               <p className="text-xs text-muted-foreground font-mono break-all">
                                 Tx: {withdrawal.txHash}
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
            <AlertDialogTitle>Confirm Withdrawal</AlertDialogTitle>
            <AlertDialogDescription>
              Please review the details below before confirming this withdrawal.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-muted-foreground">Amount</p>
                <p className="text-lg font-semibold">{amount} {asset}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Network Fee</p>
                <p className="text-lg font-semibold">{fee} {asset}</p>
              </div>
            </div>

            <div className="p-3 rounded-lg bg-green-500/10 border border-green-500/20">
              <p className="text-xs text-green-200/80">
                You will receive: <strong>{receivedAmount} {asset}</strong>
              </p>
            </div>

            <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
              <p className="text-xs text-blue-200/80">
                To: <span className="font-mono break-all">{selectedAddress?.address}</span>
              </p>
              <p className="text-xs text-blue-200/80 mt-1">
                Chain: <strong>{chain}</strong>
              </p>
            </div>
          </div>

          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                // Check if this is a new address (first withdrawal to this address)
                if (selectedAddress && !selectedAddress.isVerified) {
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
              {isSubmitting ? "Processing..." : "Confirm Withdrawal"}
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
              Security Verification Required
            </DialogTitle>
            <DialogDescription>
              This is your first withdrawal to a new address. Please enter your 2FA code to confirm.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="2fa-code">Verification Code</Label>
              <Input
                id="2fa-code"
                placeholder="Enter 6-digit code"
                value={twoFactorCode}
                onChange={(e) => setTwoFactorCode(e.target.value)}
                maxLength={6}
              />
              <p className="text-xs text-muted-foreground">
                Enter the code from your authenticator app or email
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
              Cancel
            </Button>
            <Button
              onClick={handleVerify2FAAndSubmit}
              disabled={isVerifying2FA || !twoFactorCode}
            >
              {isVerifying2FA ? "Verifying..." : "Verify & Confirm"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Withdraw;
