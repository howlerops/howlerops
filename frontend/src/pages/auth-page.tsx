import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { LoginForm } from "@/components/auth/login-form";
import { SignupForm } from "@/components/auth/signup-form";
import { OAuthButtonGroup } from "@/components/auth/oauth-button-group";
import { BiometricAuthButton } from "@/components/auth/biometric-auth-button";
import { AuthSeparator } from "@/components/auth/auth-separator";
import { HowlerOpsIcon } from "@/components/ui/howlerops-icon";
import { useAuthStore } from "@/store/auth-store";
import { shouldEnforceHostedAuth } from "@/lib/environment";

type AuthLocationState = {
  from?: {
    pathname: string;
    search?: string;
    hash?: string;
  };
};

export function AuthPage() {
  const [activeTab, setActiveTab] = useState<"login" | "signup">("login");
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const clearError = useAuthStore((state) => state.clearError);
  const location = useLocation();
  const navigate = useNavigate();
  const enforceAuth = useMemo(() => shouldEnforceHostedAuth(), []);

  const locationState = location.state as AuthLocationState | null;
  const fromLocation = locationState?.from;
  const redirectPath = useMemo(() => {
    if (!fromLocation) {
      return "/dashboard";
    }

    const search = fromLocation.search ?? "";
    const hash = fromLocation.hash ?? "";

    return `${fromLocation.pathname}${search}${hash}`;
  }, [fromLocation]);

  useEffect(() => {
    if (isAuthenticated) {
      navigate(redirectPath, { replace: true });
    }
  }, [isAuthenticated, navigate, redirectPath]);

  const handleTabChange = (value: string) => {
    setActiveTab(value as "login" | "signup");
    clearError();
  };

  const handleAuthSuccess = () => {
    navigate(redirectPath, { replace: true });
  };

  return (
    <div className="min-h-screen bg-muted/40 flex flex-col items-center justify-center px-4 py-8">
      {/* Brand Header */}
      <div className="mb-8 flex items-center justify-center gap-3">
        <HowlerOpsIcon size={80} variant="icon" />
        <h1 className="text-lg font-medium text-muted-foreground">HowlerOps</h1>
      </div>

      {/* Auth Card */}
      <Card className="w-full max-w-lg shadow-lg">
        <CardHeader className="space-y-1 text-center pb-4">
          <CardTitle className="text-2xl font-semibold">
            {activeTab === "login" ? "Welcome Back" : "Create Account"}
          </CardTitle>
          <CardDescription>
            {enforceAuth
              ? "Sign in or create an account to continue to the hosted workspace."
              : "Sign in to sync your data across devices and unlock cloud features."}
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Primary Auth: Email/Password */}
          <Tabs
            value={activeTab}
            onValueChange={handleTabChange}
            className="w-full"
          >
            <TabsList className="grid w-full grid-cols-2 mb-4">
              <TabsTrigger value="login">Login</TabsTrigger>
              <TabsTrigger value="signup">Sign Up</TabsTrigger>
            </TabsList>
            <TabsContent value="login">
              <LoginForm onSuccess={handleAuthSuccess} />
            </TabsContent>
            <TabsContent value="signup">
              <SignupForm onSuccess={handleAuthSuccess} />
            </TabsContent>
          </Tabs>

          {/* Separator */}
          <AuthSeparator />

          {/* Secondary Auth Methods */}
          <div className="space-y-3">
            <OAuthButtonGroup onSuccess={handleAuthSuccess} />
            <BiometricAuthButton onSuccess={handleAuthSuccess} />
          </div>
        </CardContent>
      </Card>

      {/* Footer Links */}
      <div className="mt-6 text-center text-sm text-muted-foreground">
        <a href="#" className="hover:text-foreground transition-colors">
          Privacy Policy
        </a>
        <span className="mx-2">•</span>
        <a href="#" className="hover:text-foreground transition-colors">
          Terms of Service
        </a>
      </div>
    </div>
  );
}
