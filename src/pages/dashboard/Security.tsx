import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Lock, Smartphone, Globe, CheckCircle2, ShieldAlert, Info, AlertTriangle } from "lucide-react";
import { toast } from "sonner";

const Security = () => {
  const { t } = useTranslation();
  const [isEnabled, setIsEnabled] = useState(false);
  const [qrCode, setQrCode] = useState("");
  const [secret, setSecret] = useState("");
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [token, setToken] = useState("");
  const [isSettingUp, setIsSettingUp] = useState(false);
  const [isLoadingStatus, setIsLoadingStatus] = useState(true);
  const [step, setStep] = useState(1); // 1: QR, 2: Backup Codes
  const [open, setOpen] = useState(false);

  const fetchStatus = async () => {
    try {
      const authToken = localStorage.getItem("token");
      const res = await fetch("/api/auth/me", {
        headers: { Authorization: `Bearer ${authToken}` }
      });
      if (res.ok) {
        const data = await res.json();
        setIsEnabled(data.twoFactorEnabled);
      }
    } catch (error) {
      console.error("Security fetchStatus error:", error);
    } finally {
      setIsLoadingStatus(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, []);

  const handleSetup = async () => {
    setIsSettingUp(true);
    try {
      const authToken = localStorage.getItem("token");
      const res = await fetch("/api/auth/2fa/setup", {
        method: "POST",
        headers: { Authorization: `Bearer ${authToken}` }
      });
      
      if (res.status === 404) {
        throw new Error("2FA Setup API endpoint not found (404)");
      }

      const data = await res.json();
      if (res.ok) {
        setQrCode(data.qrCodeUrl);
        setSecret(data.secret);
        setBackupCodes(data.backupCodes);
        setStep(1);
        setOpen(true);
      } else {
        throw new Error(data.error);
      }
    } catch (error: any) {
      console.error("2FA Setup error:", error);
      toast.error(error.message || "Failed to start 2FA setup");
    } finally {
      setIsSettingUp(false);
    }
  };

  const handleEnable = async () => {
    setIsSettingUp(true);
    console.log("Attempting to enable 2FA with token:", token);
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

      if (res.status === 404) {
        throw new Error("2FA Enable API endpoint not found (404)");
      }

      const data = await res.json();
      if (res.ok) {
        toast.success(t("dashboard.security.enabled"));
        setIsEnabled(true);
        setOpen(false);
      } else {
        throw new Error(data.error);
      }
    } catch (error: any) {
      console.error("2FA Enable error:", error);
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
                  <Button 
                    variant={isEnabled ? "outline" : "default"} 
                    size="sm" 
                    onClick={isEnabled ? undefined : handleSetup}
                    disabled={isSettingUp || isLoadingStatus}
                  >
                    {isSettingUp ? "..." : (isEnabled ? t("dashboard.security.disable2FA") : t("dashboard.security.enable2FA"))}
                  </Button>
                  <DialogContent className="sm:max-w-[425px]">
                    <DialogHeader>
                      <DialogTitle>{t("dashboard.security.setup2FA")}</DialogTitle>
                      <DialogDescription>
                        {step === 1 ? t("dashboard.security.scanQR") : "Save your backup codes in a safe place."}
                      </DialogDescription>
                    </DialogHeader>

                    {step === 1 ? (
                      <div className="flex flex-col items-center gap-6 py-4">
                        {qrCode ? (
                          <div className="p-4 bg-white rounded-xl">
                            <img src={qrCode} alt="2FA QR Code" className="w-48 h-48" />
                          </div>
                        ) : (
                          <div className="w-48 h-48 bg-secondary animate-pulse rounded-xl flex items-center justify-center text-xs text-muted-foreground" />
                        )}
                        <div className="w-full space-y-2 text-center">
                          <p className="text-xs text-muted-foreground font-medium">Secret Key (Manual):</p>
                          <code className="text-xs font-mono bg-secondary px-3 py-1 rounded-full border border-border">{secret}</code>
                        </div>
                        <Button onClick={() => setStep(2)} className="w-full">Next: Backup Codes</Button>
                      </div>
                    ) : (
                      <div className="space-y-6 py-4">
                        <div className="grid grid-cols-2 gap-2 p-4 bg-secondary/30 rounded-xl border border-border">
                          {backupCodes.map(code => (
                            <code key={code} className="text-xs font-mono text-center py-1">{code}</code>
                          ))}
                        </div>
                        <div className="flex gap-2 p-3 rounded-lg bg-blue-500/5 border border-blue-500/20">
                          <Info size={16} className="text-blue-500 shrink-0" />
                          <p className="text-[11px] text-blue-200/80 leading-relaxed">
                            These codes can be used once each to log in if you lose your device.
                          </p>
                        </div>
                        <div className="space-y-2">
                          <Label>{t("dashboard.security.enterCode")}</Label>
                          <Input 
                            placeholder="000000" 
                            value={token} 
                            onChange={(e) => setToken(e.target.value)}
                            className="text-center text-lg tracking-[0.5em] font-bold"
                            maxLength={6}
                          />
                        </div>
                        <DialogFooter>
                          <Button onClick={handleEnable} className="w-full" disabled={isSettingUp || token.length !== 6}>
                            {isSettingUp ? "Verifying..." : t("dashboard.security.verify")}
                          </Button>
                        </DialogFooter>
                      </div>
                    )}
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

import { Dialog } from "@/components/ui/dialog";

export default Security;
