import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Plus, Trash2, Check, Clock, AlertCircle } from "lucide-react";
import { toast } from "sonner";

interface WithdrawalAddress {
  id: number;
  address: string;
  addressType: "BTC" | "ETH" | "USDC" | "USDT";
  label: string;
  isVerified: boolean;
  isPrimary: boolean;
  createdAt: string;
  verifiedAt: string | null;
  deactivatedAt: string | null;
}

const Addresses = () => {
  const { t } = useTranslation();
  const [addresses, setAddresses] = useState<WithdrawalAddress[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isVerifyDialogOpen, setIsVerifyDialogOpen] = useState(false);
  const [selectedAddressId, setSelectedAddressId] = useState<number | null>(null);

  // Form states
  const [newAddress, setNewAddress] = useState("");
  const [addressType, setAddressType] = useState<"ETH" | "BTC" | "USDC" | "USDT">("ETH");
  const [label, setLabel] = useState("");
  const [verificationToken, setVerificationToken] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [deleteAddressId, setDeleteAddressId] = useState<number | null>(null);

  // Fetch addresses
  const fetchAddresses = async () => {
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/addresses", {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        setAddresses(data.addresses || []);
      } else {
        toast.error("Failed to fetch addresses");
      }
    } catch (error) {
      console.error("Fetch addresses error:", error);
      toast.error("Failed to fetch addresses");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchAddresses();
  }, []);

  // Add new address
  const handleAddAddress = async () => {
    if (!newAddress || !label) {
      toast.error("Please fill in all fields");
      return;
    }

    setIsSubmitting(true);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/addresses", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          address: newAddress,
          addressType,
          label,
        }),
      });

      const data = await res.json();
      if (res.ok) {
        toast.success(data.message || "Address added successfully");
        setNewAddress("");
        setLabel("");
        setAddressType("ETH");
        setIsDialogOpen(false);
        await fetchAddresses();
      } else {
        toast.error(data.error || "Failed to add address");
      }
    } catch (error) {
      console.error("Add address error:", error);
      toast.error("Failed to add address");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Verify address
  const handleVerifyAddress = async () => {
    if (!verificationToken || !selectedAddressId) {
      toast.error("Please enter verification token");
      return;
    }

    setIsSubmitting(true);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`/api/addresses/${selectedAddressId}/verify`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ token: verificationToken }),
      });

      const data = await res.json();
      if (res.ok) {
        toast.success(data.message || "Address verified successfully");
        setVerificationToken("");
        setIsVerifyDialogOpen(false);
        await fetchAddresses();
      } else {
        toast.error(data.error || "Failed to verify address");
      }
    } catch (error) {
      console.error("Verify address error:", error);
      toast.error("Failed to verify address");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Set as primary
  const handleSetPrimary = async (addressId: number) => {
    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`/api/addresses/${addressId}/primary`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
      });

      const data = await res.json();
      if (res.ok) {
        toast.success(data.message || "Primary address set");
        await fetchAddresses();
      } else {
        toast.error(data.error || "Failed to set primary address");
      }
    } catch (error) {
      console.error("Set primary error:", error);
      toast.error("Failed to set primary address");
    }
  };

  // Delete/Deactivate address
  const handleDeleteAddress = async () => {
    if (!deleteAddressId) return;

    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`/api/addresses/${deleteAddressId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });

      const data = await res.json();
      if (res.ok) {
        toast.success(data.message || "Address deactivated");
        setDeleteAddressId(null);
        await fetchAddresses();
      } else {
        toast.error(data.error || "Failed to deactivate address");
      }
    } catch (error) {
      console.error("Delete address error:", error);
      toast.error("Failed to deactivate address");
    }
  };

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">Withdrawal Addresses</h1>
        <Button onClick={() => setIsDialogOpen(true)} className="gap-2">
          <Plus size={20} />
          Add Address
        </Button>
      </div>

      {isLoading ? (
        <Card className="bg-card/50 border-border/50">
          <CardContent className="pt-6">
            <div className="text-center text-muted-foreground">Loading addresses...</div>
          </CardContent>
        </Card>
      ) : addresses.length === 0 ? (
        <Card className="bg-card/50 border-border/50">
          <CardContent className="pt-6">
            <div className="text-center text-muted-foreground">No addresses added yet. Click "Add Address" to get started.</div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {addresses.map((addr) => (
            <Card key={addr.id} className="bg-card/50 border-border/50">
              <CardContent className="pt-6">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 space-y-3">
                    <div className="flex items-center gap-3">
                      <h3 className="font-semibold">{addr.label}</h3>
                      <Badge variant={addr.addressType === "BTC" ? "default" : "secondary"}>
                        {addr.addressType}
                      </Badge>
                      {addr.isVerified && (
                        <Badge variant="outline" className="gap-1">
                          <Check size={14} />
                          Verified
                        </Badge>
                      )}
                      {!addr.isVerified && (
                        <Badge variant="outline" className="gap-1 text-yellow-600">
                          <Clock size={14} />
                          Pending
                        </Badge>
                      )}
                      {addr.isPrimary && (
                        <Badge className="bg-blue-600">Primary</Badge>
                      )}
                      {addr.deactivatedAt && (
                        <Badge variant="destructive">Deactivated</Badge>
                      )}
                    </div>

                    <div className="space-y-2">
                      <p className="text-sm text-muted-foreground break-all font-mono">
                        {addr.address}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        Added: {new Date(addr.createdAt).toLocaleDateString()}
                      </p>
                      {addr.verifiedAt && (
                        <p className="text-xs text-green-600">
                          Verified: {new Date(addr.verifiedAt).toLocaleDateString()}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex gap-2">
                    {!addr.isVerified && !addr.deactivatedAt && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setSelectedAddressId(addr.id);
                          setIsVerifyDialogOpen(true);
                        }}
                      >
                        Verify
                      </Button>
                    )}

                    {addr.isVerified && !addr.isPrimary && !addr.deactivatedAt && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleSetPrimary(addr.id)}
                      >
                        Set Primary
                      </Button>
                    )}

                    {!addr.deactivatedAt && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setDeleteAddressId(addr.id)}
                        className="text-destructive hover:text-destructive"
                      >
                        <Trash2 size={16} />
                      </Button>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Add Address Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Add Withdrawal Address</DialogTitle>
            <DialogDescription>
              Add a new cryptocurrency address for withdrawals. You'll receive a verification email.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="label">Address Label</Label>
              <Input
                id="label"
                placeholder="e.g., My Ethereum Wallet"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="type">Asset Type</Label>
              <Select value={addressType} onValueChange={(value: any) => setAddressType(value)}>
                <SelectTrigger id="type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="BTC">Bitcoin (BTC)</SelectItem>
                  <SelectItem value="ETH">Ethereum (ETH)</SelectItem>
                  <SelectItem value="USDC">USD Coin (USDC)</SelectItem>
                  <SelectItem value="USDT">Tether (USDT)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="address">Wallet Address</Label>
              <Input
                id="address"
                placeholder="0x1234567890abcdef1234567890abcdef12345678"
                value={newAddress}
                onChange={(e) => setNewAddress(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                {addressType === "BTC" ? "Bitcoin address (P2PKH, P2SH, or Bech32)" : "Ethereum address or ERC-20 token address"}
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsDialogOpen(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button
              onClick={handleAddAddress}
              disabled={isSubmitting || !newAddress || !label}
            >
              {isSubmitting ? "Adding..." : "Add Address"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Verify Address Dialog */}
      <Dialog open={isVerifyDialogOpen} onOpenChange={setIsVerifyDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Verify Address</DialogTitle>
            <DialogDescription>
              Enter the verification code from your email to confirm this address.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="token">Verification Code</Label>
              <Input
                id="token"
                placeholder="Paste the verification token from your email"
                value={verificationToken}
                onChange={(e) => setVerificationToken(e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsVerifyDialogOpen(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button
              onClick={handleVerifyAddress}
              disabled={isSubmitting || !verificationToken}
            >
              {isSubmitting ? "Verifying..." : "Verify"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteAddressId !== null} onOpenChange={(open) => !open && setDeleteAddressId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Deactivate Address</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently deactivate the address and you won't be able to use it for withdrawals. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteAddress}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Deactivate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default Addresses;
