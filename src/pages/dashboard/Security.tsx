import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Lock, Smartphone, Globe, CheckCircle2, ShieldAlert } from "lucide-react";
import { toast } from "sonner";

const Security = () => {
  const { t } = useTranslation();
  const [isEnabled, setIsEnabled] = useState(false);
  const [qrCode, setQrCode] = useState("");
  const [secret, setSecret] = useState("");
  const [token, setToken] = useState("");
  const [isSettingUp, setIsSettingUp] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    // 实际应从后端获取当前用户的 2FA 状态，这里简化处理，假设从 localStorage 获取部分信息
    const user = JSON.parse(localStorage.getItem("user") || "{}");
    // 提示：真实场景应调用 GET /api/auth/me
  }, []);

  const handleSetup = async () => {
    try {
      const authToken = localStorage.getItem("token");
      const res = await fetch("/api/auth/2fa/setup", {
        method: "POST",
        headers: { Authorization: `Bearer ${authToken}` }
      });
      const data = await res.json();
      setQrCode(data.qrCodeUrl);
      setSecret(data.secret);
      setOpen(true);
    } catch (error) {
      toast.error("Failed to start 2FA setup");
    }
  };

  const handleEnable = async () => {
    setIsSettingUp(true);
    try {
      const authToken = localStorage.getItem("token");
      const res = await fetch("/api/auth/2fa/enable", {
        method: "POST",
        headers: { 
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}` 
        },
        body: JSON.stringify({ token })
      });
      const data = await res.json();
      if (res.ok) {
        toast.success(t("dashboard.security.enabled"));
        setIsEnabled(true);
        setOpen(false);
      } else {
        throw new Error(data.error);
      }
    } catch (error: any) {
      toast.error(error.message);
    } finally {
      setIsSettingUp(false);
    }
  };

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">{t("dashboard.security.title")}</h1>
      </div>

      <div className="grid gap-6">
        <Card className="bg-card/50 border-border/50">
          <CardHeader>
            <div className="flex items-center gap-2">
              <Lock className="text-primary" size={20} />
              <CardTitle className="text-lg">{t("dashboard.security.password")}</CardTitle>
            </div>
            <CardDescription>Update your password to keep your account secure.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 max-w-md">
            <div className="space-y-2">
              <Label htmlFor="current-password">Current Password</Label>
              <Input id="current-password" type="password" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-password">New Password</Label>
              <Input id="new-password" type="password" />
            </div>
            <Button>{t("dashboard.security.password")}</Button>
          </CardContent>
        </Card>

        <div className="grid gap-6 md:grid-cols-2">
          {/* 2FA */}
          <Card className="bg-card/50 border-border/50">
            <CardHeader>
              <div className="flex items-center gap-2">
                <Smartphone className="text-primary" size={20} />
                <CardTitle className="text-lg">{t("dashboard.security.twoFactor")}</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                Add an extra layer of security to your account by enabling two-factor authentication.
              </p>
              
              <div className="flex items-center justify-between p-4 rounded-xl bg-secondary/30 border border-border/50">
                <div className="flex items-center gap-3">
                  {isEnabled ? (
                    <CheckCircle2 className="text-green-500" size={20} />
                  ) : (
                    <ShieldAlert className="text-muted-foreground" size={20} />
                  )}
                  <div className="text-sm font-medium">
                    Status: <span className={isEnabled ? "text-green-500" : "text-muted-foreground"}>
                      {isEnabled ? t("dashboard.security.enabled") : t("dashboard.security.disabled")}
                    </span>
                  </div>
                </div>

                <Dialog open={open} onOpenChange={setOpen}>
                  <Button variant={isEnabled ? "outline" : "default"} size="sm" onClick={isEnabled ? undefined : handleSetup}>
                    {isEnabled ? t("dashboard.security.disable2FA") : t("dashboard.security.enable2FA")}
                  </Button>
                  <DialogContent className="sm:max-w-[425px]">
                    <DialogHeader>
                      <DialogTitle>{t("dashboard.security.setup2FA")}</DialogTitle>
                      <DialogDescription>{t("dashboard.security.scanQR")}</DialogDescription>
                    </DialogHeader>
                    <div className="flex flex-col items-center gap-6 py-4">
                      {qrCode && (
                        <div className="p-4 bg-white rounded-xl">
                          <img src={qrCode} alt="2FA QR Code" className="w-48 h-48" />
                        </div>
                      )}
                      <div className="w-full space-y-2 text-center">
                        <p className="text-xs text-muted-foreground">Secret Key (Manual Entry):</p>
                        <code className="text-xs font-mono bg-secondary px-2 py-1 rounded">{secret}</code>
                      </div>
                      <div className="w-full space-y-2">
                        <Label>{t("dashboard.security.enterCode")}</Label>
                        <Input 
                          placeholder="000000" 
                          value={token} 
                          onChange={(e) => setToken(e.target.value)}
                          className="text-center text-lg tracking-[0.5em] font-bold"
                          maxLength={6}
                        />
                      </div>
                    </div>
                    <DialogFooter>
                      <Button onClick={handleEnable} className="w-full" disabled={isSettingUp || token.length !== 6}>
                        {isSettingUp ? "Verifying..." : t("dashboard.security.verify")}
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>
            </CardContent>
          </Card>

          {/* Whitelist Placeholder */}
          <Card className="bg-card/50 border-border/50">
            <CardHeader>
              <div className="flex items-center gap-2">
                <Globe className="text-primary" size={20} />
                <CardTitle className="text-lg">{t("dashboard.security.whitelist")}</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                Restrict withdrawals to only pre-approved wallet addresses.
              </p>
              <Button variant="outline" size="sm">Manage Addresses</Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Security;