import { AlertCircle, CheckCircle,Shield } from 'lucide-react';
import React, { useState } from 'react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';

interface SSOConfigFormProps {
  organizationId: string;
  onClose: () => void;
  onSuccess?: () => void;
}

export function SSOConfigForm({
  organizationId,
  onClose,
  onSuccess,
}: SSOConfigFormProps) {
  const [provider, setProvider] = useState<string>('');
  const [providerName, setProviderName] = useState<string>('');
  const [metadata, setMetadata] = useState<string>('{}');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      // Validate JSON metadata
      JSON.parse(metadata);

      const response = await fetch(`/api/organizations/${organizationId}/sso`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          provider,
          provider_name: providerName,
          metadata,
          enabled: false, // Start disabled, enable after testing
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to configure SSO');
      }

      setSuccess(true);
      setTimeout(() => {
        onSuccess?.();
        onClose();
      }, 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to configure SSO');
    } finally {
      setIsSubmitting(false);
    }
  };

  const providerTemplates: Record<string, string> = {
    saml: JSON.stringify(
      {
        idp_metadata_url: 'https://idp.example.com/metadata',
        sp_entity_id: 'sql-studio',
        sp_assertion_url: 'https://your-domain.com/api/auth/sso/saml/acs',
      },
      null,
      2
    ),
    oauth2: JSON.stringify(
      {
        client_id: 'your-client-id',
        client_secret: 'your-client-secret',
        auth_url: 'https://oauth.example.com/authorize',
        token_url: 'https://oauth.example.com/token',
        redirect_url: 'https://your-domain.com/api/auth/sso/callback',
        scopes: ['openid', 'profile', 'email'],
      },
      null,
      2
    ),
    oidc: JSON.stringify(
      {
        client_id: 'your-client-id',
        client_secret: 'your-client-secret',
        issuer: 'https://oidc.example.com',
        redirect_url: 'https://your-domain.com/api/auth/sso/callback',
      },
      null,
      2
    ),
  };

  const handleProviderChange = (value: string) => {
    setProvider(value);
    if (providerTemplates[value]) {
      setMetadata(providerTemplates[value]);
    }
  };

  return (
    <Dialog open onOpenChange={() => !isSubmitting && onClose()}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>
            <div className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Configure Single Sign-On (SSO)
            </div>
          </DialogTitle>
          <DialogDescription>
            Set up SSO for your organization to enable secure, centralized authentication
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="provider">SSO Provider Type</Label>
            <Select value={provider} onValueChange={handleProviderChange}>
              <SelectTrigger id="provider">
                <SelectValue placeholder="Select provider type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="saml">SAML 2.0</SelectItem>
                <SelectItem value="oauth2">OAuth 2.0</SelectItem>
                <SelectItem value="oidc">OpenID Connect (OIDC)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="provider-name">Provider Name</Label>
            <Select value={providerName} onValueChange={setProviderName}>
              <SelectTrigger id="provider-name">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Okta">Okta</SelectItem>
                <SelectItem value="Auth0">Auth0</SelectItem>
                <SelectItem value="AzureAD">Azure Active Directory</SelectItem>
                <SelectItem value="Google">Google Workspace</SelectItem>
                <SelectItem value="OneLogin">OneLogin</SelectItem>
                <SelectItem value="PingIdentity">Ping Identity</SelectItem>
                <SelectItem value="Custom">Custom Provider</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="metadata">Configuration (JSON)</Label>
            <Textarea
              id="metadata"
              value={metadata}
              onChange={(e) => setMetadata(e.target.value)}
              className="font-mono text-sm"
              rows={10}
              placeholder="Enter provider configuration as JSON"
            />
            <p className="text-xs text-muted-foreground">
              Configuration varies by provider type. Use the template as a guide.
            </p>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {success && (
            <Alert>
              <CheckCircle className="h-4 w-4 text-green-500" />
              <AlertDescription>
                SSO configuration saved successfully! Redirecting...
              </AlertDescription>
            </Alert>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || !provider || !providerName}>
              {isSubmitting ? 'Configuring...' : 'Configure SSO'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}