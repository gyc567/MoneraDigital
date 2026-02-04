import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { ArrowLeftRight, TrendingUp, AlertTriangle, Info, CheckCircle2 } from "lucide-react";
import { toast } from "sonner";

interface LendingPosition {
  id: number;
  asset: string;
  amount: string;
  apy: string;
  endDate: string;
  accruedYield: string;
}

const ASSETS = ["BTC", "ETH", "USDT", "USDC", "SOL"] as const;
const DURATIONS = [30, 90, 180, 360] as const;

type AssetType = typeof ASSETS[number];
type DurationType = typeof DURATIONS[number];

function getDurationMultiplier(duration: DurationType): number {
  if (duration >= 360) return 1.5;
  if (duration >= 180) return 1.25;
  if (duration >= 90) return 1.1;
  return 1.0;
}

function calculateAPY(asset: AssetType, duration: DurationType): string {
  const baseRates: Record<AssetType, number> = {
    BTC: 4.5,
    ETH: 5.2,
    USDT: 8.5,
    USDC: 8.2,
    SOL: 6.8,
  };
  const multiplier = getDurationMultiplier(duration);
  const rate = baseRates[asset] ?? 5.0;
  return (rate * multiplier).toFixed(2);
}

function calculateEstimatedYield(amount: string, apy: string, duration: DurationType): string {
  if (!amount) return "0.0000";
  const parsedAmount = parseFloat(amount);
  const parsedAPY = parseFloat(apy);
  const years = duration / 365;
  return (parsedAmount * (parsedAPY / 100) * years).toFixed(4);
}

function Lending() {
  const { t } = useTranslation();
  const [positions, setPositions] = useState<LendingPosition[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const [asset, setAsset] = useState<string>("USDT");
  const [amount, setAmount] = useState<string>("");
  const [duration, setDuration] = useState<string>("30");

  const apy = calculateAPY(asset as AssetType, parseInt(duration) as DurationType);
  const estYield = calculateEstimatedYield(amount, apy, parseInt(duration) as DurationType);

  async function fetchPositions() {
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/lending/positions", {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setPositions(data.positions || []);
      }
    } catch (error) {
      console.error("Failed to fetch positions", error);
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    fetchPositions();
  }, []);

  // Format number with 7 decimal places
  const formatNumber = (num: string | number): string => {
    if (typeof num === "string") {
      const parsed = parseFloat(num);
      if (isNaN(parsed)) return num;
      num = parsed;
    }
    const formatted = new Intl.NumberFormat("en-US", {
      minimumFractionDigits: 0,
      maximumFractionDigits: 7,
    }).format(num);
    
    return formatted.replace(/(\.\d*?[1-9])0+$/g, '$1').replace(/\.$/, '');
  };

  async function handleApply(e: React.FormEvent) {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      const token = localStorage.getItem("token");
      const res = await fetch("/api/lending/apply", {
        method: "POST",
        headers: { 
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify({ asset, amount, durationDays: duration })
      });

      if (!res.ok) throw new Error("Application failed");

      toast.success(t("dashboard.lending.success"));
      setIsDialogOpen(false);
      fetchPositions();
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "Application failed";
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t("dashboard.nav.lending")}</h1>
          <p className="text-muted-foreground">{t("dashboard.lending.applyDescription")}</p>
        </div>
        
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogTrigger asChild>
            <Button size="lg" className="gap-2 shadow-lg shadow-primary/20">
              <ArrowLeftRight size={18} />
              {t("dashboard.lending.applyTitle")}
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px] bg-card border-border">
            <DialogHeader>
              <DialogTitle>{t("dashboard.lending.applyTitle")}</DialogTitle>
              <DialogDescription>{t("dashboard.lending.applyDescription")}</DialogDescription>
            </DialogHeader>
            <form onSubmit={handleApply} className="space-y-6 py-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>{t("dashboard.lending.asset")}</Label>
                  <Select value={asset} onValueChange={setAsset}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {ASSETS.map(a => <SelectItem key={a} value={a}>{a}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>{t("dashboard.lending.duration")}</Label>
                  <Select value={duration} onValueChange={setDuration}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {DURATIONS.map(d => <SelectItem key={d} value={d.toString()}>{d} d</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              
              <div className="space-y-2">
                <Label>{t("dashboard.lending.amount")}</Label>
                <Input 
                  type="number" 
                  placeholder="0.00" 
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  required
                />
              </div>

              <div className="p-4 rounded-xl bg-secondary/50 border border-border space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("dashboard.lending.estimatedApy")}</span>
                  <span className="text-primary font-bold">{apy}%</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("dashboard.lending.estimatedYield")}</span>
                  <span className="font-mono">{estYield} {asset}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("dashboard.lending.riskLevel")}</span>
                  <Badge variant="outline" className="text-[10px] h-5 bg-blue-500/10 text-blue-400 border-blue-500/20 uppercase font-bold tracking-wider">Low-Med</Badge>
                </div>
              </div>

              <div className="flex gap-2 p-3 rounded-lg bg-orange-500/5 border border-orange-500/20">
                <AlertTriangle size={16} className="text-orange-500 shrink-0" />
                <p className="text-[11px] text-orange-200/80 leading-relaxed">
                  {t("dashboard.lending.riskWarning")}
                </p>
              </div>

              <DialogFooter>
                <Button type="submit" className="w-full" disabled={isSubmitting}>
                  {isSubmitting ? "Processing..." : t("dashboard.lending.submit")}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="grid gap-6">
        <Card className="bg-card/50 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <TrendingUp size={20} className="text-primary" />
              {t("dashboard.lending.activePositions")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="h-32 flex items-center justify-center text-muted-foreground">Loading...</div>
            ) : positions.length === 0 ? (
              <div className="h-32 flex flex-col items-center justify-center text-muted-foreground gap-2 border-2 border-dashed border-border rounded-xl">
                <Info size={24} />
                <span>No active lending positions</span>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent border-border/50">
                    <TableHead className="text-muted-foreground">ID</TableHead>
                    <TableHead className="text-muted-foreground">Asset</TableHead>
                    <TableHead className="text-muted-foreground">Amount</TableHead>
                    <TableHead className="text-muted-foreground">APY</TableHead>
                    <TableHead className="text-muted-foreground">End Date</TableHead>
                    <TableHead className="text-muted-foreground text-right">Yield</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {positions.map((pos) => (
                    <TableRow key={pos.id} className="border-border/50 hover:bg-secondary/30 transition-colors group">
                      <TableCell className="font-mono text-[10px] text-muted-foreground uppercase">#{pos.id.toString().padStart(4, '0')}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <CheckCircle2 size={14} className="text-primary" />
                          <span className="font-bold">{pos.asset}</span>
                        </div>
                      </TableCell>
                      <TableCell className="font-medium">{formatNumber(pos.amount)}</TableCell>
                      <TableCell className="text-primary font-bold">{pos.apy}%</TableCell>
                      <TableCell className="text-sm text-muted-foreground">{new Date(pos.endDate).toLocaleDateString()}</TableCell>
                      <TableCell className="text-right">
                        <div className="font-mono font-bold text-green-400">+{formatNumber(pos.accruedYield)}</div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default Lending;