import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ArrowRight, CheckCircle2, Clock, AlertCircle } from "lucide-react";
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
  createdAt: string;
  completedAt: string | null;
}

const Withdraw = () => {
  const { t } = useTranslation();
  const [addresses, setAddresses] = useState<WithdrawalAddress[]>([]);
  const [withdrawalHistory, setWithdrawalHistory] = useState<Withdrawal[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isHistoryLoading, setIsHistoryLoading] = useState(false);

  // Form states
  const [selectedAddressId, setSelectedAddressId] = useState<string>("");
  const [amount, setAmount] = useState("");
  const [asset, setAsset] = useState<"BTC" | "ETH" | "USDC" | "USDT">("ETH");
  const [isConfirming, setIsConfirming] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedAddress, setSelectedAddress] = useState<WithdrawalAddress | null>(null);

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

        // Auto-select primary address or first verified address
        const primary = verified.find((addr: WithdrawalAddress) => addr.isPrimary);
        if (primary) {
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

  useEffect(() => {
    fetchAddresses();
    fetchHistory();
  }, []);

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

                  <Button
                    onClick={() => setIsConfirming(true)}
                    disabled={!amount || Number(amount) <= 0}
                    className="w-full gap-2"
                  >
                    <ArrowRight size={16} />
                    Review & Confirm
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
                          {withdrawal.txHash && (
                            <p className="text-xs text-muted-foreground font-mono break-all">
                              Tx: {withdrawal.txHash}
                            </p>
                          )}
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
                <p className="text-xs text-muted-foreground">To Address</p>
                <p className="text-xs font-mono truncate">{selectedAddress?.address}</p>
              </div>
            </div>

            <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
              <p className="text-xs text-blue-200/80">
                This withdrawal will be processed to your verified address. A confirmation email will be sent to your account.
              </p>
            </div>
          </div>

          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleSubmitWithdrawal}
              disabled={isSubmitting}
              className="bg-green-600 hover:bg-green-700"
            >
              {isSubmitting ? "Processing..." : "Confirm Withdrawal"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default Withdraw;
