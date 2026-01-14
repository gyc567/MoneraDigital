import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface RegisterError {
  code: string;
  message: string;
  details?: string;
}

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

  const getLocalizedError = (message: string) => {
    if (!message) return "";
    const msg = message.toLowerCase();
    
    if (msg.includes("email is required")) return t("auth.errors.emailRequired");
    if (msg.includes("invalid email")) return t("auth.errors.invalidEmail");
    if (msg.includes("password is required")) return t("auth.errors.passwordRequired");
    if (msg.includes("at least 8 characters")) return t("auth.errors.passwordTooShort");
    if (msg.includes("uppercase, lowercase, and digit")) return t("auth.errors.passwordComplexity");
    if (msg.includes("email is already registered") || msg.includes("user already exists") || msg.includes("email already exists")) return t("auth.errors.userAlreadyExists");
    
    return message;
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

      let data: RegisterError;
      const contentType = res.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        data = await res.json();
      } else {
        const text = await res.text();
        console.error("Non-JSON response received:", text);
        throw new Error(t("auth.errors.registrationFailed"));
      }

      if (!res.ok) {
        handleApiError(data);
        return;
      }

      console.log("Registration successful");
      toast.success(t("auth.register.successMessage"));
      
      setTimeout(() => {
        navigate("/login");
      }, 1000);
    } catch (error: any) {
      console.error("Registration error:", error);
      if (!emailError && !passwordError) {
        toast.error(error.message);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleApiError = (data: RegisterError) => {
    const errorCode = data.code || "";
    const errorMessage = data.message || "";
    const errorDetails = data.details || "";
    const localizedMessage = getLocalizedError(errorMessage);

    switch (errorCode) {
      case "VALIDATION_ERROR":
        if (errorDetails === "email") {
          setEmailError(t("auth.errors.invalidEmailFormat"));
        } else if (errorDetails === "password") {
          setPasswordError(t("auth.errors.invalidPasswordFormat"));
        } else {
          toast.error(t("auth.errors.invalidParameters"));
        }
        break;
      case "EMAIL_ALREADY_EXISTS":
        setEmailError(t("auth.errors.emailAlreadyRegistered"));
        break;
      case "INTERNAL_ERROR":
      case "PANIC_RECOVERED":
        toast.error(t("auth.errors.serverError"));
        break;
      default:
        if (errorDetails === "email") {
          setEmailError(localizedMessage);
        } else if (errorDetails === "password") {
          setPasswordError(localizedMessage);
        } else {
          toast.error(localizedMessage || t("auth.errors.registrationFailed"));
        }
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
