import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts';
import { cn } from "@/lib/utils";
import { ArrowRight } from "lucide-react";
import { Link } from "react-router-dom";

const assetData = [
  { name: 'BTC', value: 2450000, color: '#F7931A' },
  { name: 'ETH', value: 1560000, color: '#627EEA' },
  { name: 'USDT', value: 850000, color: '#26A17B' },
  { name: 'SOL', value: 260432.84, color: '#14F195' },
];

const Assets = () => {
  const { t } = useTranslation();

  const assets = [
    { currency: "BTC", total: "38.12", available: "35.00", frozen: "3.12", value: "$2,450,000.00", source: t("dashboard.assets.owned") },
    { currency: "ETH", total: "457.18", available: "400.00", frozen: "57.18", value: "$1,560,000.00", source: t("dashboard.assets.owned") },
    { currency: "USDT", total: "850,000.00", available: "500,000.00", frozen: "350,000.00", value: "$850,000.00", source: t("dashboard.assets.borrowed") },
    { currency: "SOL", total: "1,850.40", available: "1,850.40", frozen: "0.00", value: "$260,432.84", source: t("dashboard.assets.owned") },
  ];

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">{t("dashboard.nav.assets")}</h1>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {/* Distribution Chart */}
        <Card className="lg:col-span-1 bg-card/50 border-border/50">
          <CardHeader>
            <CardTitle className="text-base font-semibold">{t("dashboard.overview.assetDistribution")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={assetData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {assetData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip 
                    formatter={(value: number) => `$${value.toLocaleString()}`}
                    contentStyle={{ backgroundColor: 'hsl(var(--card))', borderColor: 'hsl(var(--border))' }}
                  />
                  <Legend verticalAlign="bottom" height={36}/>
                </PieChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* Detailed List */}
        <Card className="lg:col-span-2 bg-card/50 border-border/50">
          <CardHeader>
            <CardTitle className="text-base font-semibold">Asset Breakdown</CardTitle>
          </CardHeader>
          <CardContent>
             <Table>
               <TableHeader>
                 <TableRow className="hover:bg-transparent border-border/50">
                   <TableHead className="text-muted-foreground">{t("dashboard.assets.currency")}</TableHead>
                   <TableHead className="text-muted-foreground">{t("dashboard.assets.total")}</TableHead>
                   <TableHead className="text-muted-foreground">{t("dashboard.assets.available")}</TableHead>
                   <TableHead className="text-muted-foreground text-right">Value (USD)</TableHead>
                   <TableHead className="text-muted-foreground text-right">Actions</TableHead>
                 </TableRow>
               </TableHeader>
               <TableBody>
                 {assets.map((asset) => (
                   <TableRow key={asset.currency} className="border-border/50 hover:bg-secondary/30 transition-colors">
                     <TableCell className="font-medium">
                       <div className="flex items-center gap-2">
                         <div className="w-2 h-2 rounded-full" style={{ backgroundColor: assetData.find(d => d.name === asset.currency)?.color }} />
                         {asset.currency}
                       </div>
                     </TableCell>
                     <TableCell>{asset.total}</TableCell>
                     <TableCell>{asset.available}</TableCell>
                     <TableCell className="text-right font-semibold">{asset.value}</TableCell>
                     <TableCell className="text-right">
                       {parseFloat(asset.available.replace(/,/g, '')) > 0 && (
                         <Button asChild size="sm" variant="outline" className="gap-1">
                           <Link to={`/dashboard/withdraw?asset=${asset.currency}`}>
                             <ArrowRight size={14} />
                             Withdraw
                           </Link>
                         </Button>
                       )}
                     </TableCell>
                   </TableRow>
                 ))}
               </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default Assets;
