import { useState, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { PiggyBank, TrendingUp, Info, ArrowRight, Clock, Percent, ShieldCheck, RefreshCw, History, AlertTriangle, Wallet, Filter } from "lucide-react";
import { toast } from "sonner";

interface Product {
  id: number;
  name: string;
  currency: string;
  apy: number;
  duration: number;
  minAmount: number;
  maxAmount: number;
  totalQuota: number;
  autoRenew: boolean;
}

interface WealthOrder {
  id: number;
  productTitle: string;
  currency: string;
  amount: string;
  interestExpected: string;
  interestPaid: string;
  interestAccrued: string;
  startDate: string;
  endDate: string;
  autoRenew: boolean;
  status: number;
}

interface Asset {
  currency: string;
  total: string;
  available: string;
}

interface PaginationProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

const Pagination = ({ currentPage, totalPages, onPageChange }: PaginationProps) => {
  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-center gap-2 mt-4">
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage <= 1}
      >
        <ChevronLeft size={16} />
      </Button>
      <span className="text-sm text-muted-foreground px-2">
        {currentPage} / {totalPages}
      </span>
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage >= totalPages}
      >
        <ChevronRight size={16} />
      </Button>
    </div>
  );
};

const FixedDeposit = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [amount, setAmount] = useState("");
  const [autoRenew, setAutoRenew] = useState(false);
  const [activeTab, setActiveTab] = useState("products");

  const [orders, setOrders] = useState<WealthOrder[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [selectedOrder, setSelectedOrder] = useState<WealthOrder | null>(null);
  const [redeemDialogOpen, setRedeemDialogOpen] = useState(false);
  const [redeemAmount, setRedeemAmount] = useState("");
  const [isRedeeming, setIsRedeeming] = useState(false);

  const [assets, setAssets] = useState<Asset[]>([]);
  const [isLoadingAssets, setIsLoadingAssets] = useState(true);
  const [currencyFilter, setCurrencyFilter] = useState<string | null>(null);

  const [productsPage, setProductsPage] = useState(1);
  const [productsTotal, setProductsTotal] = useState(0);
  const [ordersPage, setOrdersPage] = useState(1);
  const [ordersTotal, setOrdersTotal] = useState(0);

  const fetchAssets = async () => {
    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }
      const res = await fetch("/api/assets", {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.status === 401 || res.status === 403) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
        return;
      }
      if (res.ok) {
        const data = await res.json();
        setAssets(Array.isArray(data) ? data : []);
      }
    } catch (error) {
      console.error("Failed to fetch assets", error);
      toast.error(t("auth.errors.networkError"));
    } finally {
      setIsLoadingAssets(false);
    }
  };

  const fetchOrders = async () => {
    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }
      const res = await fetch(`/api/wealth/orders?page=${ordersPage}&pageSize=20`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.status === 401 || res.status === 403) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
        return;
      }
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data.data)) {
          setOrders(data.data);
          setOrdersTotal(data.total || 0);
        } else if (Array.isArray(data)) {
          setOrders(data);
          setOrdersTotal(data.length);
        }
      }
    } catch (error) {
      console.error("Failed to fetch orders", error);
      toast.error(t("auth.errors.networkError"));
    }
  };

  const fetchProducts = async () => {
    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }
      const res = await fetch(`/api/wealth/products?page=${productsPage}&pageSize=10`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.status === 401 || res.status === 403) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
        return;
      }
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data.data)) {
          const transformed = data.data.map((p: any) => ({
            id: p.id,
            name: p.title,
            currency: p.currency,
            apy: p.apy,
            duration: p.duration,
            minAmount: parseFloat(p.minAmount),
            maxAmount: parseFloat(p.maxAmount),
            totalQuota: parseFloat(p.remainingQuota),
            autoRenew: p.autoRenewAllowed
          }));
          setProducts(transformed);
          setProductsTotal(data.total || 0);
        } else if (Array.isArray(data)) {
          const transformed = data.map((p: any) => ({
            id: p.id,
            name: p.title,
            currency: p.currency,
            apy: p.apy,
            duration: p.duration,
            minAmount: parseFloat(p.minAmount),
            maxAmount: parseFloat(p.maxAmount),
            totalQuota: parseFloat(p.remainingQuota),
            autoRenew: p.autoRenewAllowed
          }));
          setProducts(transformed);
          setProductsTotal(data.length);
        }
      }
    } catch (error) {
      console.error("Failed to fetch products", error);
    }
  };

  useEffect(() => {
    fetchAssets();
  }, []);

  useEffect(() => {
    if (assets.length > 0) {
      fetchProducts();
    } else {
      setProducts([]);
      setProductsTotal(0);
    }
  }, [assets.length]);

  useEffect(() => {
    fetchOrders();
  }, [ordersPage]);

  const displayAssets = assets || [];
  const displayProducts = products || [];
  const displayOrders = orders || [];

  const allCurrencies = useMemo(() => {
    const productCurrencies = displayProducts.map(p => p.currency);
    const orderCurrencies = displayOrders.map(o => o.currency);
    const assetCurrencies = displayAssets.map(a => a.currency);
    return [...new Set([...productCurrencies, ...orderCurrencies, ...assetCurrencies])].sort();
  }, [displayAssets, displayOrders, displayProducts]);

  useEffect(() => {
    if (!currencyFilter && allCurrencies.length > 0) {
      setCurrencyFilter(allCurrencies[0]);
    }
  }, [allCurrencies, currencyFilter]);

  const filteredProducts = useMemo(() => {
    if (!currencyFilter) return displayProducts;
    return displayProducts.filter(p => p.currency === currencyFilter);
  }, [displayProducts, currencyFilter]);

  const filteredOrders = useMemo(() => {
    if (!currencyFilter) return displayOrders;
    return displayOrders.filter(o => o.currency === currencyFilter);
  }, [displayOrders, currencyFilter]);

  const handleSubscribe = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProduct || !amount) return;

    setIsSubmitting(true);

    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }

      const res = await fetch("/api/wealth/subscribe", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
          productId: selectedProduct.id,
          amount: amount,
          autoRenew: autoRenew
        })
      });

      if (res.status === 401 || res.status === 403) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
        return;
      }

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        const friendlyError = getFriendlyErrorMessage(errorData.error, t);
        throw new Error(friendlyError);
      }

      const data = await res.json();
      toast.success(t("dashboard.fixedDeposit.subscribeSuccess"));
      if (data.interestExpected) {
        toast.info(`${t("dashboard.fixedDeposit.interest")}: ${data.interestExpected} ${selectedProduct.currency}`);
      }
      setOpen(false);
      setAmount("");
      setAutoRenew(false);
      setSelectedProduct(null);
      fetchAssets();
      fetchOrders();
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : t("dashboard.fixedDeposit.subscribeFailed");
      toast.error(errorMessage);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getFriendlyErrorMessage = (errorKey: string, t: any): string => {
    const errorMap: Record<string, string> = {
      "insufficient balance": t("auth.errors.wealth.insufficientBalance"),
      "product not found": t("auth.errors.wealth.productNotFound"),
      "order not found": t("auth.errors.wealth.orderNotFound"),
      "product not available": t("auth.errors.wealth.productNotAvailable"),
      "amount below minimum": t("auth.errors.wealth.amountBelowMin"),
      "amount above maximum": t("auth.errors.wealth.amountAboveMax"),
      "quota exceeded": t("auth.errors.wealth.quotaExceeded"),
      "order already redeemed": t("auth.errors.wealth.orderAlreadyRedeemed"),
      "invalid redemption type": t("auth.errors.wealth.invalidRedemptionType"),
      "failed to fetch price": t("auth.errors.wealth.priceFetchFailed"),
      "product is not active": t("auth.errors.wealth.productNotAvailable"),
    };
    return errorMap[errorKey] || errorKey || t("auth.errors.unknownError");
  };

  const handleRedeem = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedOrder) return;

    setIsRedeeming(true);
    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }
      const res = await fetch("/api/wealth/redeem", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
          orderId: selectedOrder.id,
          redemptionType: ""
        })
      });

      if (res.status === 401 || res.status === 403) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
        return;
      }

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        const friendlyError = getFriendlyErrorMessage(errorData.error, t);
        throw new Error(friendlyError);
      }

      toast.success(t("dashboard.fixedDepositHistory.redeemSuccess"));
      setRedeemDialogOpen(false);
      setRedeemAmount("");
      fetchOrders();
      fetchAssets();
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : t("auth.errors.unknownError");
      toast.error(errorMessage);
    } finally {
      setIsRedeeming(false);
    }
  };

  const formatNumber = (num: number | string): string => {
    if (typeof num === "string") {
      num = parseFloat(num);
    }
    if (isNaN(num) || !isFinite(num)) {
      return "0";
    }
    return new Intl.NumberFormat("en-US", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 8,
    }).format(num);
  };

  const formatDateTime = (date: Date) => {
    return date.toLocaleString("en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const formatDateTimeRows = (dateStr: string) => {
    const date = new Date(dateStr);
    const dateStrFormatted = date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    });
    const timeStr = date.toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
    });
    return { date: dateStrFormatted, time: timeStr };
  };

  const calculateInterest = (): string => {
    if (!selectedProduct || !amount) return "0.00";
    const principal = parseFloat(amount);
    if (isNaN(principal) || principal <= 0) return "0.00";
    const interest = (principal * (selectedProduct.apy / 100) * selectedProduct.duration) / 365;
    if (isNaN(interest) || !isFinite(interest)) return "0.00";
    return formatNumber(interest);
  };

  const STATUS_MAP: Record<number, { label: string; variant: "default" | "secondary" | "outline" | "destructive" }> = {
    0: { label: "pending_interest", variant: "secondary" },
    1: { label: "accruing_interest", variant: "default" },
    2: { label: "matured", variant: "outline" },
    3: { label: "redeemed", variant: "outline" },
    4: { label: "renewing", variant: "secondary" },
  };

  const validateAmount = (): { valid: boolean; error?: string } => {
    if (!selectedProduct || !amount) {
      return { valid: true };
    }
    const amountNum = parseFloat(amount);
    if (isNaN(amountNum) || amountNum <= 0) {
      return { valid: false, error: t("dashboard.fixedDeposit.invalidAmount") };
    }
    if (amountNum < selectedProduct.minAmount) {
      return { valid: false, error: `${t("dashboard.fixedDeposit.amountBelowMinError")} ${formatNumber(selectedProduct.minAmount)} ${selectedProduct.currency}` };
    }
    if (amountNum > selectedProduct.maxAmount) {
      return { valid: false, error: `${t("dashboard.fixedDeposit.amountAboveMaxError")} ${formatNumber(selectedProduct.maxAmount)} ${selectedProduct.currency}` };
    }
    return { valid: true };
  };

  const getStatusInfo = (status: number) => {
    const statusConfig = STATUS_MAP[status];
    if (!statusConfig) {
      return { label: t("dashboard.fixedDepositHistory.statusUnknown"), variant: "outline" as const };
    }
    const labelKey = `dashboard.fixedDepositHistory.status_${statusConfig.label}`;
    return {
      label: t(labelKey),
      variant: statusConfig.variant
    };
  };

  const getDaysRemaining = (endDate: string) => {
    const end = new Date(endDate);
    const now = new Date();
    const diff = end.getTime() - now.getTime();
    const days = Math.ceil(diff / (1000 * 60 * 60 * 24));
    return days > 0 ? days : 0;
  };

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between gap-4">
          <Tabs value={activeTab} onValueChange={setActiveTab} className="w-auto">
            <TabsList className="grid w-full grid-cols-2 h-12">
              <TabsTrigger value="products" className="text-base gap-2 px-6">
                <TrendingUp size={18} />
                {t("dashboard.fixedDeposit.products")}
              </TabsTrigger>
              <TabsTrigger value="history" className="text-base gap-2 px-6">
                <History size={18} />
                {t("dashboard.fixedDeposit.history")}
              </TabsTrigger>
            </TabsList>
          </Tabs>
          
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{t("dashboard.fixedDeposit.pageTitle")}</h1>
            <p className="text-muted-foreground">{t("dashboard.fixedDeposit.pageDesc")}</p>
          </div>
        </div>
        
        <div className="w-full">
          {isLoadingAssets ? (
            <span className="text-sm text-muted-foreground">{t("dashboard.fixedDeposit.loading")}</span>
          ) : (
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm text-muted-foreground">{t("dashboard.fixedDeposit.filterByCurrency")}:</span>
              {displayAssets.length === 0 ? (
                <span className="text-sm text-muted-foreground">{t("dashboard.fixedDeposit.noAssets")}</span>
              ) : (
                displayAssets.map((asset) => (
                  <Button
                    key={asset.currency}
                    variant={currencyFilter === asset.currency ? "default" : "outline"}
                    size="sm"
                    onClick={() => setCurrencyFilter(currencyFilter === asset.currency ? null : asset.currency)}
                    className="gap-1"
                  >
                    <Wallet size={14} />
                    {asset.currency}
                  </Button>
                ))
              )}
              {currencyFilter && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setCurrencyFilter(null)}
                  className="text-muted-foreground"
                >
                  {t("dashboard.fixedDeposit.clearFilter")}
                </Button>
              )}
            </div>
          )}
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsContent value="products" className="mt-0 space-y-6">
          <div className="grid gap-4">
            <Card className="bg-card/50 border-border/50">
              <CardHeader>
                <div className="flex items-center justify-between gap-4">
                  <div className="flex items-center gap-2">
                    <TrendingUp size={20} className="text-primary" />
                    <CardTitle className="text-lg">{t("dashboard.fixedDeposit.availableProducts")}</CardTitle>
                  </div>
                  <div className="flex items-center gap-3">
                    {currencyFilter && (
                      <span className="text-sm text-muted-foreground font-mono">
                        {formatNumber(assets.find(a => a.currency === currencyFilter)?.available || 0)} {currencyFilter}
                      </span>
                    )}
                    <Dialog open={open} onOpenChange={setOpen}>
                      <DialogTrigger asChild>
                        <Button size="lg" className="gap-2 shadow-lg shadow-primary/20" disabled={!selectedProduct}>
                          <PiggyBank size={18} />
                          {t("dashboard.fixedDeposit.subscribe")}
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="sm:max-w-[500px] bg-card border-border">
                        <DialogHeader>
                          <DialogTitle>{selectedProduct?.name || t("dashboard.fixedDeposit.confirmSubscribe")}</DialogTitle>
                          <DialogDescription>{t("dashboard.fixedDeposit.subscribeDesc")}</DialogDescription>
                        </DialogHeader>
                        <form onSubmit={handleSubscribe} className="space-y-6 py-4">
                          <div className="p-4 rounded-xl bg-secondary/50 border border-border space-y-3">
                            <div className="flex justify-between text-sm">
                              <span className="text-muted-foreground">{t("dashboard.fixedDeposit.productName")}</span>
                              <span className="font-medium">{selectedProduct?.name}</span>
                            </div>
                            <div className="flex justify-between text-sm">
                              <span className="text-muted-foreground">{t("dashboard.fixedDeposit.apy")}</span>
                              <span className="text-primary font-bold">{selectedProduct?.apy}%</span>
                            </div>
                            <div className="flex justify-between text-sm">
                              <span className="text-muted-foreground">{t("dashboard.fixedDeposit.duration")}</span>
                              <span className="font-medium">{selectedProduct?.duration} {t("dashboard.fixedDeposit.days")}</span>
                            </div>
                          </div>

                          <div className="space-y-2">
                            <Label>{t("dashboard.fixedDeposit.amount")} ({selectedProduct?.currency})</Label>
                            {selectedProduct && (
                              <div className="flex items-center justify-between text-sm p-3 rounded-lg bg-primary/5 border border-primary/20">
                                <span className="text-muted-foreground flex items-center gap-1.5">
                                  <Wallet size={16} className="text-primary" />
                                  {t("dashboard.fixedDeposit.available")}
                                </span>
                                <span className="font-mono font-bold text-primary">
                                  {formatNumber(assets.find(a => a.currency === selectedProduct.currency)?.available || 0)} {selectedProduct.currency}
                                </span>
                              </div>
                            )}
                            <Input
                              type="number"
                              placeholder={`${t("dashboard.fixedDeposit.minAmount")}: ${selectedProduct?.minAmount || 0} ${selectedProduct?.currency || 'USDT'}`}
                              value={amount}
                              onChange={(e) => setAmount(e.target.value)}
                              required
                             />
                             {selectedProduct && amount && parseFloat(amount) < selectedProduct.minAmount && (
                               <p className="text-xs text-red-500">
                                 {t("dashboard.fixedDeposit.amountBelowMinError")} {selectedProduct.minAmount} {selectedProduct.currency}
                               </p>
                             )}
                             {selectedProduct && amount && parseFloat(amount) > selectedProduct.maxAmount && (
                               <p className="text-xs text-red-500">
                                 {t("dashboard.fixedDeposit.amountAboveMaxError")} {selectedProduct.maxAmount} {selectedProduct.currency}
                               </p>
                             )}
                          </div>

                          <div className="p-4 rounded-xl bg-primary/5 border border-primary/20 space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-muted-foreground">{t("dashboard.fixedDeposit.interest")}</span>
                              <span className="font-bold text-primary">{calculateInterest()} {selectedProduct?.currency}</span>
                            </div>
                            <div className="flex justify-between text-xs text-muted-foreground">
                              <span>{t("dashboard.fixedDeposit.maturityDate")}</span>
                              <span>
                                {selectedProduct
                                  ? formatDateTime(new Date(Date.now() + selectedProduct.duration * 24 * 60 * 60 * 1000))
                                  : "-"}
                              </span>
                            </div>
                          </div>

                          {selectedProduct?.autoRenew && (
                            <div className="flex items-center justify-between p-4 rounded-lg bg-secondary/50 border border-border">
                              <div className="flex items-center gap-3">
                                <RefreshCw size={18} className="text-primary" />
                                <div className="flex flex-col">
                                  <span className="text-sm font-medium">{t("dashboard.fixedDeposit.autoRenewTitle")}</span>
                                  <span className="text-xs text-muted-foreground">{t("dashboard.fixedDeposit.autoRenewDesc")}</span>
                                </div>
                              </div>
                              <Switch
                                checked={autoRenew}
                                onCheckedChange={setAutoRenew}
                              />
                            </div>
                          )}

                          <div className="flex gap-2 p-3 rounded-lg bg-blue-500/5 border border-blue-500/20">
                            <ShieldCheck size={16} className="text-blue-500 shrink-0" />
                            <p className="text-xs text-muted-foreground">
                              {t("dashboard.fixedDeposit.securityNotice")}
                            </p>
                          </div>

                           <DialogFooter>
                             <Button type="button" variant="outline" onClick={() => setOpen(false)}>{t("common.cancel")}</Button>
                             <Button 
                               type="submit" 
                               className="w-full" 
                               disabled={
                                 isSubmitting || 
                                 !amount || 
                                 parseFloat(amount) < (selectedProduct?.minAmount || 0) ||
                                 parseFloat(amount) > (selectedProduct?.maxAmount || 0)
                               }
                             >
                              {isSubmitting ? t("dashboard.fixedDeposit.subscribing") : t("dashboard.fixedDeposit.confirmSubscribe")}
                            </Button>
                          </DialogFooter>
                        </form>
                      </DialogContent>
                    </Dialog>
                  </div>
                </div>
                <CardDescription>{t("dashboard.fixedDeposit.productDesc")}</CardDescription>
              </CardHeader>
              <CardContent>
                 {filteredProducts.length === 0 ? (
                   <div className="h-48 flex flex-col items-center justify-center text-muted-foreground gap-3">
                     <Info size={32} className="text-muted-foreground/50" />
                     <span className="text-lg">{t("dashboard.fixedDeposit.noProducts")}</span>
                     {currencyFilter && (
                       <span className="text-sm">
                         {t("dashboard.fixedDeposit.tryOtherCurrency")}
                       </span>
                     )}
                   </div>
                 ) : (
                   <>
                     <Table>
                       <TableHeader>
                         <TableRow className="hover:bg-transparent border-border/50">
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.productName")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.currency")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.apy")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.duration")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.minAmount")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.maxAmount")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.autoRenew")}</TableHead>
                           <TableHead className="text-muted-foreground text-right">{t("dashboard.fixedDeposit.action")}</TableHead>
                         </TableRow>
                       </TableHeader>
                       <TableBody>
                         {filteredProducts.map((product) => (
                           <TableRow 
                             key={product.id} 
                             className="border-border/50 hover:bg-secondary/30 transition-colors group cursor-pointer"
                             onClick={() => setSelectedProduct(product)}
                           >
                             <TableCell>
                               <div className="flex items-center gap-2">
                                 <PiggyBank size={16} className="text-primary" />
                                 <span className="font-medium">{product.name}</span>
                               </div>
                             </TableCell>
                             <TableCell>
                               <Badge variant="secondary" className="font-mono">{product.currency}</Badge>
                             </TableCell>
                             <TableCell>
                               <Badge variant="outline" className="bg-primary/10 text-primary border-primary/20 font-bold">
                                 {product.apy}%
                               </Badge>
                             </TableCell>
                             <TableCell>
                               <div className="flex items-center gap-1 text-sm">
                                 <Clock size={14} className="text-muted-foreground" />
                                 <span>{product.duration} {t("dashboard.fixedDeposit.days")}</span>
                               </div>
                             </TableCell>
                             <TableCell className="font-mono text-sm">{formatNumber(product.minAmount)} {product.currency}</TableCell>
                             <TableCell className="font-mono text-sm">{formatNumber(product.maxAmount)} {product.currency}</TableCell>
                             <TableCell>
                               {product.autoRenew ? (
                                 <Badge variant="default" className="gap-1">
                                   <RefreshCw size={12} />
                                   {t("dashboard.fixedDeposit.support")}
                                 </Badge>
                               ) : (
                                 <Badge variant="outline" className="text-muted-foreground">
                                   {t("dashboard.fixedDeposit.notSupport")}
                                 </Badge>
                               )}
                              </TableCell>
                              <TableCell className="text-right">
                                <Button 
                                  size="sm" 
                                  variant={selectedProduct?.id === product.id ? "default" : "ghost"}
                                  className="gap-1"
                                >
                                  {selectedProduct?.id === product.id ? t("dashboard.fixedDeposit.selected") : t("dashboard.fixedDeposit.select")}
                                  <ArrowRight size={14} />
                                </Button>
                              </TableCell>
                           </TableRow>
                         ))}
                       </TableBody>
                     </Table>
                     {filteredProducts.length > 0 && (
                       <Pagination 
                         currentPage={productsPage} 
                         totalPages={Math.max(1, Math.ceil(productsTotal / 10))} 
                         onPageChange={setProductsPage}
                       />
                     )}
                   </>
                  )}
               </CardContent>
            </Card>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card className="bg-gradient-to-br from-primary/10 to-transparent border-primary/20">
                <CardContent className="pt-6">
                  <div className="flex items-center gap-2 mb-2">
                    <ShieldCheck className="text-primary" size={20} />
                    <span className="font-medium">{t("dashboard.fixedDeposit.security")}</span>
                  </div>
                  <p className="text-sm text-muted-foreground">{t("dashboard.fixedDeposit.securityDesc")}</p>
                </CardContent>
              </Card>
              
              <Card className="bg-gradient-to-br from-green-500/10 to-transparent border-green-500/20">
                <CardContent className="pt-6">
                  <div className="flex items-center gap-2 mb-2">
                    <Percent className="text-green-500" size={20} />
                    <span className="font-medium">{t("dashboard.fixedDeposit.stableReturns")}</span>
                  </div>
                  <p className="text-sm text-muted-foreground">{t("dashboard.fixedDeposit.stableReturnsDesc")}</p>
                </CardContent>
              </Card>
              
              <Card className="bg-gradient-to-br from-blue-500/10 to-transparent border-blue-500/20">
                <CardContent className="pt-6">
                  <div className="flex items-center gap-2 mb-2">
                    <Info className="text-blue-500" size={20} />
                    <span className="font-medium">{t("dashboard.fixedDeposit.flexibleTerms")}</span>
                  </div>
                  <p className="text-sm text-muted-foreground">{t("dashboard.fixedDeposit.flexibleTermsDesc")}</p>
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="history" className="mt-0 space-y-6">
          <div className="grid gap-6">
            <Card className="bg-card/50 border-border/50">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <History size={20} className="text-primary" />
                  {t("dashboard.fixedDepositHistory.title")}
                </CardTitle>
                <CardDescription>{t("dashboard.fixedDepositHistory.description")}</CardDescription>
              </CardHeader>
              <CardContent>
                 {filteredOrders.length === 0 ? (
                   <div className="h-32 flex flex-col items-center justify-center text-muted-foreground gap-2 border-2 border-dashed border-border rounded-xl">
                     <Info size={24} />
                     <span>{t("dashboard.fixedDepositHistory.noOrders")}</span>
                   </div>
                 ) : (
                   <>
                     <Table>
                       <TableHeader>
                         <TableRow className="hover:bg-transparent border-border/50">
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.orderNo")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDeposit.productName")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.amount")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.expectedReturn")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.paidReturn")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.accruedReturn")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.period")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.maturity")}</TableHead>
                           <TableHead className="text-muted-foreground">{t("dashboard.fixedDepositHistory.status")}</TableHead>
                           <TableHead className="text-muted-foreground text-right">{t("dashboard.fixedDepositHistory.action")}</TableHead>
                         </TableRow>
                       </TableHeader>
                       <TableBody>
                         {filteredOrders.map((order) => {
                           const statusInfo = getStatusInfo(order.status);
                           const daysRemaining = order.status === 1 ? getDaysRemaining(order.endDate) : null;
                           return (
                             <TableRow key={order.id} className="border-border/50 hover:bg-secondary/30 transition-colors group">
                               <TableCell className="font-mono text-xs text-muted-foreground">
                                 #{order.id.toString().padStart(6, '0')}
                               </TableCell>
                               <TableCell>
                                 <div className="flex items-center gap-2">
                                   <PiggyBank size={14} className="text-primary" />
                                   <span className="font-medium">{order.productTitle}</span>
                                 </div>
                               </TableCell>
                               <TableCell className="font-mono">{formatNumber(order.amount)} {order.currency}</TableCell>
                               <TableCell className="font-mono text-primary">{formatNumber(order.interestExpected)} {order.currency}</TableCell>
                               <TableCell className="font-mono text-green-400">{formatNumber(order.interestPaid)} {order.currency}</TableCell>
                               <TableCell className="font-mono font-bold text-green-400">+{formatNumber(order.interestAccrued)} {order.currency}</TableCell>
                               <TableCell>
                                 <div className="flex flex-col text-sm">
                                   {(() => {
                                     const start = formatDateTimeRows(order.startDate);
                                     const end = formatDateTimeRows(order.endDate);
                                     return (
                                       <>
                                         <span>{start.date}</span>
                                         <span className="text-xs text-muted-foreground">{start.time}</span>
                                       </>
                                     );
                                   })()}
                                 </div>
                               </TableCell>
                               <TableCell>
                                 <div className="flex flex-col">
                                   {(() => {
                                     const end = formatDateTimeRows(order.endDate);
                                     return (
                                       <>
                                         <span className="text-sm">{end.date}</span>
                                         <span className="text-xs text-muted-foreground">{end.time}</span>
                                         {daysRemaining > 0 && (
                                           <span className="text-xs text-muted-foreground">{t("dashboard.fixedDepositHistory.remainingDays")} {daysRemaining} {t("dashboard.fixedDepositHistory.days")}</span>
                                         )}
                                       </>
                                     );
                                   })()}
                                 </div>
                               </TableCell>
                               <TableCell>
                                 <Badge variant={statusInfo.variant} className="w-full justify-center">{statusInfo.label}</Badge>
                               </TableCell>
                               <TableCell className="text-right">
                                 {order.status === 1 && (
                                   <Button size="sm" variant="destructive" className="gap-1" onClick={() => { setSelectedOrder(order); setRedeemAmount(order.amount.toString()); setRedeemDialogOpen(true); }}>
                                     <RefreshCw size={12} />{t("dashboard.fixedDepositHistory.redeem")}
                                   </Button>
                                 )}
                                 {order.status === 2 && (
                                   <Button size="sm" variant="default" className="gap-1" onClick={() => { setSelectedOrder(order); setRedeemAmount(order.amount.toString()); setRedeemDialogOpen(true); }}>
                                     <ArrowRight size={12} />{t("dashboard.fixedDepositHistory.claim")}
                                   </Button>
                                 )}
                               </TableCell>
                             </TableRow>
                           );
                         })}
                       </TableBody>
                     </Table>
                     {filteredOrders.length > 0 && (
                       <Pagination 
                         currentPage={ordersPage} 
                         totalPages={Math.max(1, Math.ceil(ordersTotal / 20))} 
                         onPageChange={setOrdersPage}
                       />
                     )}
                   </>
                   )}
                 </CardContent>
            </Card>

            {currencyFilter ? (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card className="bg-gradient-to-br from-green-500/10 to-transparent border-green-500/20">
                  <CardContent className="pt-6">
                    <div className="flex items-center gap-2 mb-2">
                      <TrendingUp className="text-green-500" size={20} />
                      <span className="font-medium">{t("dashboard.fixedDepositHistory.totalPaidReturn")} ({currencyFilter})</span>
                    </div>
                    <p className="text-2xl font-bold text-green-400">+{formatNumber(filteredOrders.reduce((sum, o) => sum + parseFloat(o.interestPaid.toString()), 0))} {currencyFilter}</p>
                    <p className="text-sm text-muted-foreground">{t("dashboard.fixedDepositHistory.totalPaidReturnDesc")}</p>
                  </CardContent>
                </Card>

                <Card className="bg-gradient-to-br from-primary/10 to-transparent border-primary/20">
                  <CardContent className="pt-6">
                    <div className="flex items-center gap-2 mb-2">
                      <PiggyBank className="text-primary" size={20} />
                      <span className="font-medium">{t("dashboard.fixedDepositHistory.principal")} ({currencyFilter})</span>
                    </div>
                    <p className="text-2xl font-bold">{formatNumber(filteredOrders.filter(o => o.status === 1).reduce((sum, o) => sum + parseFloat(o.amount.toString()), 0))} {currencyFilter}</p>
                    <p className="text-sm text-muted-foreground">{t("dashboard.fixedDepositHistory.principalDesc")}</p>
                  </CardContent>
                </Card>

                <Card className="bg-gradient-to-br from-blue-500/10 to-transparent border-blue-500/20">
                  <CardContent className="pt-6">
                    <div className="flex items-center gap-2 mb-2">
                      <History className="text-blue-500" size={20} />
                      <span className="font-medium">{t("dashboard.fixedDepositHistory.completedOrders")} ({currencyFilter})</span>
                    </div>
                    <p className="text-2xl font-bold">{filteredOrders.filter(o => o.status === 3).length} {t("dashboard.fixedDepositHistory.orders")}</p>
                    <p className="text-sm text-muted-foreground">{t("dashboard.fixedDepositHistory.completedOrdersDesc")}</p>
                  </CardContent>
                </Card>
              </div>
            ) : (
              <Card className="bg-card/50 border-border/50">
                <CardContent className="pt-6">
                  <div className="flex flex-col items-center justify-center gap-2 text-muted-foreground">
                    <Info size={24} />
                    <span>{t("dashboard.fixedDepositHistory.selectCurrencyHint")}</span>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={redeemDialogOpen} onOpenChange={setRedeemDialogOpen}>
        <DialogContent className="sm:max-w-[500px] bg-card border-border">
          <DialogHeader>
            <DialogTitle>{selectedOrder?.status === 2 ? t("dashboard.fixedDepositHistory.claim") : t("dashboard.fixedDepositHistory.redeem")}</DialogTitle>
            <DialogDescription>{selectedOrder?.productTitle} - {formatNumber(selectedOrder?.amount || 0)} {selectedOrder?.currency}</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleRedeem} className="space-y-6 py-4">
            {selectedOrder?.status === 1 && (
              <div className="flex gap-2 p-3 rounded-lg bg-red-500/10 border border-red-500/30">
                <AlertTriangle size={16} className="text-red-500 shrink-0" />
                <p className="text-xs text-red-400">{t("dashboard.fixedDepositHistory.earlyRedeemWarning")}</p>
              </div>
            )}

            <div className="p-4 rounded-xl bg-secondary/50 border border-border space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t("dashboard.fixedDepositHistory.amount")}</span>
                <span className="font-medium">{formatNumber(selectedOrder?.amount || 0)} {selectedOrder?.currency}</span>
              </div>
              {selectedOrder?.status === 2 ? (
                <>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">{t("dashboard.fixedDepositHistory.accruedReturn")}</span>
                    <span className="font-medium text-green-400">+{formatNumber(selectedOrder?.interestAccrued || 0)} {selectedOrder?.currency}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">{t("dashboard.fixedDepositHistory.paidReturn")}</span>
                    <span className="font-medium text-green-400">+{formatNumber(selectedOrder?.interestPaid || 0)} {selectedOrder?.currency}</span>
                  </div>
                </>
              ) : (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("dashboard.fixedDepositHistory.expectedReturn")}</span>
                  <span className="text-muted-foreground">{t("dashboard.fixedDepositHistory.noReturn")}</span>
                </div>
              )}
              <div className="border-t border-border pt-2 mt-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground font-medium">{selectedOrder?.status === 2 ? t("dashboard.fixedDepositHistory.estimatedAmount") : t("dashboard.fixedDepositHistory.redeemAmount")}</span>
                  <span className={`font-bold ${selectedOrder?.status === 2 ? "text-primary" : "text-foreground"}`}>
                    {selectedOrder?.status === 2 ? (
                      <>{formatNumber((selectedOrder?.amount || 0) + (selectedOrder?.interestAccrued || 0) + (selectedOrder?.interestPaid || 0))} {selectedOrder?.currency}</>
                    ) : (
                      <><span className="text-muted-foreground">{formatNumber(selectedOrder?.amount || 0)} {selectedOrder?.currency} ({t("dashboard.fixedDepositHistory.principalOnly")})</span></>
                    )}
                  </span>
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setRedeemDialogOpen(false)}>{t("common.cancel")}</Button>
              <Button type="submit" className="w-full" disabled={isRedeeming}>
                {isRedeeming ? t("dashboard.fixedDepositHistory.redeeming") : (selectedOrder?.status === 2 ? t("dashboard.fixedDepositHistory.confirmClaim") : t("dashboard.fixedDepositHistory.confirmRedeem"))}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default FixedDeposit;
