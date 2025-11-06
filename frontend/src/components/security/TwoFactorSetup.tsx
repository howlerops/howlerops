import React, { useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Shield,
  Smartphone,
  Key,
  Copy,
  CheckCircle,
  AlertCircle,
  Lock,
  RefreshCw,
} from 'lucide-react';

interface TwoFactorSetupProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

interface SetupData {
  secret: string;
  qr_code: string;
  backup_codes: string[];
}

export function TwoFactorSetup({ isOpen, onClose, onSuccess }: TwoFactorSetupProps) {
  const [step, setStep] = useState<'setup' | 'verify' | 'backup'>('setup');
  const [setupData, setSetupData] = useState<SetupData | null>(null);
  const [verificationCode, setVerificationCode] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedSecret, setCopiedSecret] = useState(false);
  const [copiedCodes, setCopiedCodes] = useState(false);

  const initSetup = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/auth/2fa/enable', {
        method: 'POST',
      });

      if (!response.ok) {
        throw new Error('Failed to initialize 2FA setup');
      }

      const data = await response.json();
      setSetupData(data);
      setStep('setup');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to setup 2FA');
    } finally {
      setIsLoading(false);
    }
  };

  const verifySetup = async () => {
    if (!verificationCode || verificationCode.length !== 6) {
      setError('Please enter a 6-digit code');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/auth/2fa/confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: verificationCode }),
      });

      if (!response.ok) {
        throw new Error('Invalid verification code');
      }

      setStep('backup');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setIsLoading(false);
    }
  };

  const copyToClipboard = (text: string, type: 'secret' | 'codes') => {
    navigator.clipboard.writeText(text);
    if (type === 'secret') {
      setCopiedSecret(true);
      setTimeout(() => setCopiedSecret(false), 2000);
    } else {
      setCopiedCodes(true);
      setTimeout(() => setCopiedCodes(false), 2000);
    }
  };

  const handleComplete = () => {
    onSuccess?.();
    onClose();
  };

  React.useEffect(() => {
    if (isOpen && !setupData) {
      initSetup();
    }
  }, [isOpen, setupData]);

  return (
    <Dialog open={isOpen} onOpenChange={() => !isLoading && onClose()}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Set Up Two-Factor Authentication
          </DialogTitle>
          <DialogDescription>
            Add an extra layer of security to your account
          </DialogDescription>
        </DialogHeader>

        {step === 'setup' && setupData && (
          <div className="space-y-4">
            <Tabs defaultValue="app" className="w-full">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="app">
                  <Smartphone className="mr-2 h-4 w-4" />
                  Authenticator App
                </TabsTrigger>
                <TabsTrigger value="manual">
                  <Key className="mr-2 h-4 w-4" />
                  Manual Entry
                </TabsTrigger>
              </TabsList>

              <TabsContent value="app" className="space-y-4">
                <div className="text-center">
                  <p className="mb-4 text-sm text-muted-foreground">
                    Scan this QR code with your authenticator app
                  </p>
                  <div className="inline-block rounded-lg bg-white p-4">
                    <QRCodeSVG value={setupData.qr_code} size={200} />
                  </div>
                  <p className="mt-4 text-xs text-muted-foreground">
                    Use apps like Google Authenticator, Authy, or 1Password
                  </p>
                </div>
              </TabsContent>

              <TabsContent value="manual" className="space-y-4">
                <div className="space-y-2">
                  <Label>Secret Key</Label>
                  <div className="flex gap-2">
                    <Input
                      value={setupData.secret}
                      readOnly
                      className="font-mono text-sm"
                    />
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => copyToClipboard(setupData.secret, 'secret')}
                    >
                      {copiedSecret ? (
                        <CheckCircle className="h-4 w-4" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Enter this key manually in your authenticator app
                  </p>
                </div>
              </TabsContent>
            </Tabs>

            <div className="space-y-2">
              <Label htmlFor="verify-code">Enter Verification Code</Label>
              <Input
                id="verify-code"
                type="text"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
                value={verificationCode}
                onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, ''))}
                placeholder="000000"
                className="text-center text-2xl tracking-widest"
              />
              <p className="text-xs text-muted-foreground">
                Enter the 6-digit code from your authenticator app
              </p>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <DialogFooter>
              <Button variant="outline" onClick={onClose} disabled={isLoading}>
                Cancel
              </Button>
              <Button
                onClick={verifySetup}
                disabled={isLoading || verificationCode.length !== 6}
              >
                {isLoading ? 'Verifying...' : 'Verify & Continue'}
              </Button>
            </DialogFooter>
          </div>
        )}

        {step === 'backup' && setupData && (
          <div className="space-y-4">
            <Alert>
              <Lock className="h-4 w-4" />
              <AlertTitle>Save Your Backup Codes</AlertTitle>
              <AlertDescription>
                Store these codes in a safe place. You can use them to access your account if
                you lose your authenticator device.
              </AlertDescription>
            </Alert>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Backup Codes</CardTitle>
                <CardDescription>Each code can only be used once</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-2">
                  {setupData.backup_codes.map((code, index) => (
                    <div
                      key={index}
                      className="rounded-md bg-muted px-3 py-2 font-mono text-sm"
                    >
                      {code}
                    </div>
                  ))}
                </div>
                <Button
                  variant="outline"
                  className="mt-4 w-full"
                  onClick={() =>
                    copyToClipboard(setupData.backup_codes.join('\n'), 'codes')
                  }
                >
                  {copiedCodes ? (
                    <>
                      <CheckCircle className="mr-2 h-4 w-4" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy className="mr-2 h-4 w-4" />
                      Copy All Codes
                    </>
                  )}
                </Button>
              </CardContent>
            </Card>

            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                These backup codes will not be shown again. Make sure to save them now!
              </AlertDescription>
            </Alert>

            <DialogFooter>
              <Button onClick={handleComplete} className="w-full">
                <CheckCircle className="mr-2 h-4 w-4" />
                I've Saved My Backup Codes
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

interface TwoFactorStatus {
  enabled: boolean;
  backup_codes_count?: number;
}

export function TwoFactorStatus() {
  const [status, setStatus] = useState<TwoFactorStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showSetup, setShowSetup] = useState(false);
  const [_showDisable, setShowDisable] = useState(false);

  React.useEffect(() => {
    fetchStatus();
  }, []);

  const fetchStatus = async () => {
    try {
      const response = await fetch('/api/auth/2fa/status');
      if (response.ok) {
        const data = await response.json();
        setStatus(data);
      }
    } catch (err) {
      console.error('Failed to fetch 2FA status:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const _handleDisable = async (password: string) => {
    try {
      const response = await fetch('/api/auth/2fa/disable', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });

      if (response.ok) {
        setStatus({ ...status, enabled: false });
        setShowDisable(false);
      }
    } catch (err) {
      console.error('Failed to disable 2FA:', err);
    }
  };

  if (isLoading) return null;

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Two-Factor Authentication
            </span>
            {status?.enabled && (
              <Badge variant="default" className="bg-green-500">
                <CheckCircle className="mr-1 h-3 w-3" />
                Enabled
              </Badge>
            )}
          </CardTitle>
          <CardDescription>
            Secure your account with time-based one-time passwords
          </CardDescription>
        </CardHeader>
        <CardContent>
          {status?.enabled ? (
            <div className="space-y-4">
              <div className="text-sm text-muted-foreground">
                <p>Two-factor authentication is currently enabled.</p>
                {(status?.backup_codes_count ?? 0) > 0 && (
                  <p className="mt-2">
                    You have {status?.backup_codes_count} backup codes remaining.
                  </p>
                )}
              </div>
              <div className="flex gap-2">
                <Button variant="outline" size="sm">
                  <RefreshCw className="mr-2 h-4 w-4" />
                  Regenerate Backup Codes
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowDisable(true)}
                  className="text-destructive"
                >
                  Disable 2FA
                </Button>
              </div>
            </div>
          ) : (
            <Button onClick={() => setShowSetup(true)}>
              <Shield className="mr-2 h-4 w-4" />
              Enable Two-Factor Authentication
            </Button>
          )}
        </CardContent>
      </Card>

      {showSetup && (
        <TwoFactorSetup
          isOpen={showSetup}
          onClose={() => setShowSetup(false)}
          onSuccess={() => {
            setShowSetup(false);
            fetchStatus();
          }}
        />
      )}
    </>
  );
}
