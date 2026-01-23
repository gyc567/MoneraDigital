import { useState, useEffect } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { validateRedirectPath } from "@/lib/redirect-validator";
import { cn } from "@/lib/utils";

import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from "@/components/ui/input-otp";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");
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

  const clearErrors = () => {
    setEmailError("");
    setPasswordError("");
  };

  const isValidEmail = (email: string) => {
    const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
    return emailRegex.test(email);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearErrors();
    setIsLoading(true);

    if (!isValidEmail(email)) {
      setEmailError(t("auth.errors.invalidEmailFormat"));
      setIsLoading(false);
      return;
    }

    if (password.length < 1) {
      setPasswordError(t("auth.errors.passwordRequired"));
      setIsLoading(false);
      return;
    }

    try {
      if (requires2FA) {
        // Step 2: Verify 2FA
        const res = await fetch("/api/auth/2fa/verify-login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ userId: tempUserId, token: twoFactorToken }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || data.message);

        localStorage.setItem("token", data.access_token || data.token);
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
      if (!res.ok) {
        const errorCode = data.code || "";
        
        if (errorCode === "EMAIL_NOT_FOUND") {
          setEmailError(t("auth.errors.emailNotFound"));
        } else if (errorCode === "INVALID_CREDENTIALS") {
          setEmailError(t("auth.errors.invalidEmailOrPassword"));
          setPasswordError(t("auth.errors.invalidEmailOrPassword"));
        } else {
          toast.error(t("auth.errors.serverError"));
        }
        setIsLoading(false);
        return;
      }

      if (data.requires2FA) {
        setRequires2FA(true);
        setTempUserId(data.userId);
        toast.info(t("dashboard.security.enterCode"));
        return;
      }

      localStorage.setItem("token", data.access_token || data.token);
      localStorage.setItem("user", JSON.stringify(data.user));
      toast.success(t("auth.login.successMessage"));
      // Redirect to validated returnTo or default to /dashboard
      const returnTo = validateRedirectPath((location.state as any)?.returnTo);
      navigate(returnTo);
    } catch (error: any) {
      console.error("Login error:", error);
      if (!emailError && !passwordError) {
        toast.error(t("auth.errors.serverError"));
      }
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
                  <Label htmlFor="email" className={cn(emailError && "text-red-500")}>
                    {t("auth.login.email")}
                  </Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder={t("auth.login.emailPlaceholder")}
                    value={email}
                    onChange={(e) => {
                      setEmail(e.target.value);
                      if (emailError) setEmailError("");
                    }}
                    onInvalid={(e) => {
                      e.preventDefault();
                      if (!isValidEmail(email)) {
                        setEmailError(t("auth.errors.invalidEmailFormat"));
                      }
                    }}
                    className={cn(emailError && "border-red-500 focus-visible:ring-red-500")}
                  />
                  {emailError && (
                    <p className="text-xs text-red-500 animate-in fade-in-0 slide-in-from-top-1">
                      {emailError}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password" className={cn(passwordError && "text-red-500")}>
                    {t("auth.login.password")}
                  </Label>
                  <Input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => {
                      setPassword(e.target.value);
                      if (passwordError) setPasswordError("");
                    }}
                    className={cn(passwordError && "border-red-500 focus-visible:ring-red-500")}
                  />
                  {passwordError && (
                    <p className="text-xs text-red-500 animate-in fade-in-0 slide-in-from-top-1">
                      {passwordError}
                    </p>
                  )}
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
