import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

export default function Register() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();
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

  const validatePassword = (pwd: string) => {
    // Basic client-side check to give immediate feedback if desired,
    // but primarily we rely on server response.
    // Here we just clear error if it looks okay-ish to avoid sticky errors.
    if (pwd.length >= 8) {
       // Ideally check regex too, but server is authority.
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearErrors();
    setIsLoading(true);
    console.log("Attempting registration for:", email);

    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      let data;
      const contentType = res.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        data = await res.json();
      } else {
        const text = await res.text();
        console.error("Non-JSON response received:", text);
        throw new Error(t("auth.errors.registrationFailed"));
      }

      if (!res.ok) {
        // Handle specific field errors
        if (data.details === "email") {
            setEmailError(data.message);
        } else if (data.details === "password") {
            setPasswordError(data.message);
        }
        
        throw new Error(data.error || data.message || t("auth.errors.registrationFailed"));
      }

      console.log("Registration successful");
      toast.success(t("auth.register.successMessage"));
      
      setTimeout(() => {
        navigate("/login");
      }, 1000);
    } catch (error: any) {
      console.error("Registration error:", error);
      // We already set specific field errors above if applicable.
      // Show generic toast as well for visibility.
      toast.error(error.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t("auth.register.title")}</CardTitle>
          <CardDescription>{t("auth.register.description")}</CardDescription>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email" className={cn(emailError && "text-red-500")}>
                {t("auth.register.email")}
              </Label>
              <Input
                id="email"
                type="email"
                placeholder={t("auth.register.emailPlaceholder")}
                value={email}
                onChange={(e) => {
                    setEmail(e.target.value);
                    if (emailError) setEmailError("");
                }}
                className={cn(emailError && "border-red-500 focus-visible:ring-red-500")}
                required
              />
              {emailError && (
                  <p className="text-xs text-red-500 animate-in fade-in-0 slide-in-from-top-1">
                      {emailError}
                  </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="password" className={cn(passwordError && "text-red-500")}>
                {t("auth.register.password")}
              </Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => {
                    setPassword(e.target.value);
                    if (passwordError) setPasswordError("");
                    validatePassword(e.target.value);
                }}
                className={cn(passwordError && "border-red-500 focus-visible:ring-red-500")}
                required
              />
              <p className={cn(
                  "text-xs transition-colors duration-200",
                  passwordError ? "text-red-500 font-medium" : "text-muted-foreground"
              )}>
                {passwordError || t("auth.register.passwordRequirements")}
              </p>
            </div>
          </CardContent>
          <CardFooter className="flex flex-col space-y-4">
            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? t("auth.register.registering") : t("auth.register.button")}
            </Button>
            <div className="text-sm text-center">
              {t("auth.register.hasAccount")}{" "}
              <Link to="/login" className="text-blue-600 hover:underline">
                {t("auth.register.login")}
              </Link>
            </div>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}