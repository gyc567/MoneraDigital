import { useState, useEffect } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { validateRedirectPath } from "@/lib/redirect-validator";

import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from "@/components/ui/input-otp";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [twoFactorToken, setTwoFactorToken] = useState("");
  const [requires2FA, setRequires2FA] = useState(false);
  const [tempUserId, setTempUserId] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation();

  useEffect(() => {
    if (localStorage.getItem("token")) {
      navigate("/");
    }
  }, [navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      if (requires2FA) {
        // Step 2: Verify 2FA
        const res = await fetch("/api/auth/2fa/verify-login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ userId: tempUserId, token: twoFactorToken }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error);

        localStorage.setItem("token", data.token);
        localStorage.setItem("user", JSON.stringify(data.user));
        toast.success(t("auth.login.successMessage"));
        // Redirect to validated returnTo or default to /dashboard
        const returnTo = validateRedirectPath((location.state as any)?.returnTo);
        navigate(returnTo);
        return;
      }

      // Step 1: Password Login
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      const data = await res.json();
      if (!res.ok) throw new Error(data.error || data.message || t("auth.errors.loginFailed"));

      if (data.requires2FA) {
        setRequires2FA(true);
        setTempUserId(data.userId);
        toast.info(t("dashboard.security.enterCode"));
        return;
      }

      localStorage.setItem("token", data.token);
      localStorage.setItem("user", JSON.stringify(data.user));
      toast.success(t("auth.login.successMessage"));
      // Redirect to validated returnTo or default to /dashboard
      const returnTo = validateRedirectPath((location.state as any)?.returnTo);
      navigate(returnTo);
    } catch (error: any) {
      toast.error(error.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100 dark:bg-black">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{requires2FA ? t("dashboard.security.twoFactor") : t("auth.login.title")}</CardTitle>
          <CardDescription>
            {requires2FA ? t("dashboard.security.enterCode") : t("auth.login.description")}
          </CardDescription>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            {!requires2FA ? (
              <>
                <div className="space-y-2">
                  <Label htmlFor="email">{t("auth.login.email")}</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder={t("auth.login.emailPlaceholder")}
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password">{t("auth.login.password")}</Label>
                  <Input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
              </>
            ) : (
              <div className="flex flex-col items-center space-y-4 py-4">
                <InputOTP
                  maxLength={6}
                  value={twoFactorToken}
                  onChange={(value) => setTwoFactorToken(value)}
                  autoFocus
                >
                  <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                  </InputOTPGroup>
                  <InputOTPSeparator />
                  <InputOTPGroup>
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                  </InputOTPGroup>
                </InputOTP>
              </div>
            )}
          </CardContent>
          <CardFooter className="flex flex-col space-y-4">
            <Button type="submit" className="w-full" disabled={isLoading || (requires2FA && twoFactorToken.length !== 6)}>
              {isLoading ? t("auth.login.loggingIn") : (requires2FA ? t("dashboard.security.verify") : t("auth.login.button"))}
            </Button>
            {!requires2FA && (
              <div className="text-sm text-center">
                {t("auth.login.noAccount")}{" "}
                <Link to="/register" className="text-blue-600 hover:underline">
                  {t("auth.login.register")}
                </Link>
              </div>
            )}
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}