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
    <div className="min-h-screen bg-muted/40 flex items-center justify-center px-4">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader className="space-y-1 text-center">
          <CardTitle className="text-2xl font-semibold">
            Welcome to Howlerops
          </CardTitle>
          <CardDescription>
            {enforceAuth
              ? "Sign in or create an account to continue to the hosted workspace."
              : "Sign in to sync your data across devices and unlock cloud features."}
          </CardDescription>
        </CardHeader>
        <CardContent>
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
        </CardContent>
      </Card>
    </div>
  );
}
