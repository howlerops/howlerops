import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Loader2, CheckCircle, XCircle, Eye } from 'lucide-react';

interface WhiteLabelConfig {
  organization_id: string;
  custom_domain?: string;
  logo_url?: string;
  favicon_url?: string;
  primary_color?: string;
  secondary_color?: string;
  accent_color?: string;
  company_name?: string;
  support_email?: string;
  custom_css?: string;
  hide_branding: boolean;
}

interface DomainVerification {
  id: string;
  domain: string;
  verified: boolean;
  dns_record_type: string;
  dns_record_name: string;
  dns_record_value: string;
}

export default function WhiteLabelingPage() {
  const [_config, setConfig] = useState<WhiteLabelConfig | null>(null);
  const [domains, setDomains] = useState<DomainVerification[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);

  // Form state
  const [formData, setFormData] = useState({
    logo_url: '',
    favicon_url: '',
    primary_color: '#1E40AF',
    secondary_color: '#64748B',
    accent_color: '#8B5CF6',
    company_name: '',
    support_email: '',
    custom_css: '',
    hide_branding: false,
  });

  const organizationId = 'current'; // Replace with actual org ID from context

  useEffect(() => {
    loadConfig();
    loadDomains();
  }, []);

  const loadConfig = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/organizations/${organizationId}/white-label`);
      if (response.ok) {
        const data = await response.json();
        setConfig(data);
        setFormData({
          logo_url: data.logo_url || '',
          favicon_url: data.favicon_url || '',
          primary_color: data.primary_color || '#1E40AF',
          secondary_color: data.secondary_color || '#64748B',
          accent_color: data.accent_color || '#8B5CF6',
          company_name: data.company_name || '',
          support_email: data.support_email || '',
          custom_css: data.custom_css || '',
          hide_branding: data.hide_branding || false,
        });
      }
    } catch {
      setError('Failed to load white-label configuration');
    } finally {
      setLoading(false);
    }
  };

  const loadDomains = async () => {
    try {
      const response = await fetch(`/api/organizations/${organizationId}/domains`);
      if (response.ok) {
        const data = await response.json();
        setDomains(data || []);
      }
    } catch (err) {
      console.error('Failed to load domains', err);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      setError(null);
      setSuccess(null);

      const response = await fetch(`/api/organizations/${organizationId}/white-label`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      });

      if (!response.ok) {
        throw new Error('Failed to save configuration');
      }

      const data = await response.json();
      setConfig(data);
      setSuccess('White-label configuration saved successfully!');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save configuration');
    } finally {
      setSaving(false);
    }
  };

  const handleAddDomain = async (domain: string) => {
    try {
      const response = await fetch(`/api/organizations/${organizationId}/domains`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ domain }),
      });

      if (!response.ok) {
        throw new Error('Failed to add domain');
      }

      await loadDomains();
      setSuccess('Domain added! Please follow DNS verification instructions.');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add domain');
    }
  };

  const handleVerifyDomain = async (domain: string) => {
    try {
      const response = await fetch(
        `/api/organizations/${organizationId}/domains/${domain}/verify`,
        { method: 'POST' }
      );

      if (!response.ok) {
        throw new Error('Domain verification failed. Please check DNS records.');
      }

      await loadDomains();
      setSuccess('Domain verified successfully!');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to verify domain');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 max-w-6xl">
      <div className="mb-6">
        <h1 className="text-3xl font-bold">White-labeling</h1>
        <p className="text-gray-600">Customize the appearance and branding of your SQL Studio instance</p>
      </div>

      {error && (
        <Alert variant="destructive" className="mb-4">
          <XCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {success && (
        <Alert className="mb-4">
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="branding" className="space-y-4">
        <TabsList>
          <TabsTrigger value="branding">Branding</TabsTrigger>
          <TabsTrigger value="domains">Custom Domains</TabsTrigger>
          <TabsTrigger value="advanced">Advanced</TabsTrigger>
        </TabsList>

        {/* Branding Tab */}
        <TabsContent value="branding" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card>
              <CardHeader>
                <CardTitle>Visual Branding</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <Label htmlFor="company_name">Company Name</Label>
                  <Input
                    id="company_name"
                    value={formData.company_name}
                    onChange={(e) => setFormData({ ...formData, company_name: e.target.value })}
                    placeholder="Your Company Name"
                  />
                </div>

                <div>
                  <Label htmlFor="logo_url">Logo URL</Label>
                  <Input
                    id="logo_url"
                    value={formData.logo_url}
                    onChange={(e) => setFormData({ ...formData, logo_url: e.target.value })}
                    placeholder="https://example.com/logo.png"
                  />
                  <p className="text-sm text-gray-500 mt-1">Recommended: PNG or SVG, max 200x50px</p>
                </div>

                <div>
                  <Label htmlFor="favicon_url">Favicon URL</Label>
                  <Input
                    id="favicon_url"
                    value={formData.favicon_url}
                    onChange={(e) => setFormData({ ...formData, favicon_url: e.target.value })}
                    placeholder="https://example.com/favicon.ico"
                  />
                  <p className="text-sm text-gray-500 mt-1">Recommended: 32x32px ICO or PNG</p>
                </div>

                <div>
                  <Label htmlFor="support_email">Support Email</Label>
                  <Input
                    id="support_email"
                    type="email"
                    value={formData.support_email}
                    onChange={(e) => setFormData({ ...formData, support_email: e.target.value })}
                    placeholder="support@yourcompany.com"
                  />
                </div>

                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    id="hide_branding"
                    checked={formData.hide_branding}
                    onChange={(e) => setFormData({ ...formData, hide_branding: e.target.checked })}
                    className="rounded"
                  />
                  <Label htmlFor="hide_branding">Hide "Powered by SQL Studio" branding</Label>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Color Scheme</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <Label htmlFor="primary_color">Primary Color</Label>
                  <div className="flex space-x-2">
                    <Input
                      id="primary_color"
                      value={formData.primary_color}
                      onChange={(e) => setFormData({ ...formData, primary_color: e.target.value })}
                      placeholder="#1E40AF"
                    />
                    <input
                      type="color"
                      value={formData.primary_color}
                      onChange={(e) => setFormData({ ...formData, primary_color: e.target.value })}
                      className="w-12 h-10 rounded border"
                    />
                  </div>
                </div>

                <div>
                  <Label htmlFor="secondary_color">Secondary Color</Label>
                  <div className="flex space-x-2">
                    <Input
                      id="secondary_color"
                      value={formData.secondary_color}
                      onChange={(e) => setFormData({ ...formData, secondary_color: e.target.value })}
                      placeholder="#64748B"
                    />
                    <input
                      type="color"
                      value={formData.secondary_color}
                      onChange={(e) => setFormData({ ...formData, secondary_color: e.target.value })}
                      className="w-12 h-10 rounded border"
                    />
                  </div>
                </div>

                <div>
                  <Label htmlFor="accent_color">Accent Color</Label>
                  <div className="flex space-x-2">
                    <Input
                      id="accent_color"
                      value={formData.accent_color}
                      onChange={(e) => setFormData({ ...formData, accent_color: e.target.value })}
                      placeholder="#8B5CF6"
                    />
                    <input
                      type="color"
                      value={formData.accent_color}
                      onChange={(e) => setFormData({ ...formData, accent_color: e.target.value })}
                      className="w-12 h-10 rounded border"
                    />
                  </div>
                </div>

                {/* Color Preview */}
                <div className="mt-4 p-4 border rounded">
                  <p className="text-sm font-medium mb-2">Color Preview</p>
                  <div className="flex space-x-2">
                    <div
                      className="w-16 h-16 rounded"
                      style={{ backgroundColor: formData.primary_color }}
                      title="Primary"
                    />
                    <div
                      className="w-16 h-16 rounded"
                      style={{ backgroundColor: formData.secondary_color }}
                      title="Secondary"
                    />
                    <div
                      className="w-16 h-16 rounded"
                      style={{ backgroundColor: formData.accent_color }}
                      title="Accent"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setShowPreview(!showPreview)}>
              <Eye className="w-4 h-4 mr-2" />
              {showPreview ? 'Hide Preview' : 'Show Preview'}
            </Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </Button>
          </div>
        </TabsContent>

        {/* Custom Domains Tab */}
        <TabsContent value="domains">
          <Card>
            <CardHeader>
              <CardTitle>Custom Domains</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                {domains.map((domain) => (
                  <div key={domain.id} className="flex items-center justify-between p-4 border rounded">
                    <div className="flex-1">
                      <p className="font-medium">{domain.domain}</p>
                      {domain.verified ? (
                        <div className="flex items-center text-green-600 text-sm">
                          <CheckCircle className="w-4 h-4 mr-1" />
                          Verified
                        </div>
                      ) : (
                        <div className="text-sm text-gray-600">
                          <p>Add this DNS record:</p>
                          <code className="text-xs bg-gray-100 p-1 rounded">
                            {domain.dns_record_type}: {domain.dns_record_name} = {domain.dns_record_value}
                          </code>
                        </div>
                      )}
                    </div>
                    {!domain.verified && (
                      <Button onClick={() => handleVerifyDomain(domain.domain)}>Verify</Button>
                    )}
                  </div>
                ))}
              </div>

              <div className="flex space-x-2">
                <Input
                  id="new_domain"
                  placeholder="app.yourcompany.com"
                />
                <Button
                  onClick={() => {
                    const input = document.getElementById('new_domain') as HTMLInputElement;
                    if (input.value) {
                      handleAddDomain(input.value);
                      input.value = '';
                    }
                  }}
                >
                  Add Domain
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Advanced Tab */}
        <TabsContent value="advanced">
          <Card>
            <CardHeader>
              <CardTitle>Custom CSS</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={formData.custom_css}
                onChange={(e) => setFormData({ ...formData, custom_css: e.target.value })}
                placeholder="/* Add custom CSS here */"
                rows={15}
                className="font-mono text-sm"
              />
              <p className="text-sm text-gray-500 mt-2">
                Add custom CSS to further customize the appearance. Use with caution.
              </p>
              <div className="flex justify-end mt-4">
                <Button onClick={handleSave} disabled={saving}>
                  {saving ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    'Save CSS'
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Live Preview */}
      {showPreview && (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>Live Preview</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="border rounded-lg p-6" style={{
              backgroundColor: '#ffffff',
            }}>
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center space-x-2">
                  {formData.logo_url ? (
                    <img src={formData.logo_url} alt="Logo" className="h-8" />
                  ) : (
                    <div className="h-8 w-32 bg-gray-200 rounded" />
                  )}
                </div>
                <div className="text-sm text-gray-600">
                  {formData.company_name || 'Your Company'}
                </div>
              </div>

              <div className="space-y-2">
                <button
                  className="px-4 py-2 rounded text-white"
                  style={{ backgroundColor: formData.primary_color }}
                >
                  Primary Button
                </button>
                <button
                  className="px-4 py-2 rounded text-white ml-2"
                  style={{ backgroundColor: formData.secondary_color }}
                >
                  Secondary Button
                </button>
                <button
                  className="px-4 py-2 rounded text-white ml-2"
                  style={{ backgroundColor: formData.accent_color }}
                >
                  Accent Button
                </button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
