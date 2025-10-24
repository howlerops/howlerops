import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Key,
  Plus,
  Trash2,
  Copy,
  CheckCircle,
  AlertCircle,
  Clock,
  Shield,
  Eye,
  EyeOff,
} from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  permissions: string[];
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
  revoked_at?: string;
}

interface APIKeyManagerProps {
  organizationId?: string;
}

export function APIKeyManager({ organizationId }: APIKeyManagerProps) {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchAPIKeys();
  }, []);

  const fetchAPIKeys = async () => {
    try {
      const response = await fetch('/api/api-keys');
      if (!response.ok) throw new Error('Failed to fetch API keys');
      const data = await response.json();
      setKeys(data);
    } catch (err) {
      setError('Failed to load API keys');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRevoke = async (keyId: string) => {
    if (!confirm('Are you sure you want to revoke this API key? This action cannot be undone.')) {
      return;
    }

    try {
      const response = await fetch(`/api/api-keys/${keyId}`, {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('Failed to revoke API key');

      setKeys(keys.map(k =>
        k.id === keyId ? { ...k, revoked_at: new Date().toISOString() } : k
      ));
    } catch (err) {
      setError('Failed to revoke API key');
    }
  };

  const getKeyStatus = (key: APIKey) => {
    if (key.revoked_at) return { label: 'Revoked', variant: 'destructive' as const };
    if (key.expires_at && new Date(key.expires_at) < new Date()) {
      return { label: 'Expired', variant: 'secondary' as const };
    }
    return { label: 'Active', variant: 'default' as const };
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Key className="h-5 w-5" />
              API Keys
            </CardTitle>
            <CardDescription>
              Manage programmatic access to your SQL Studio account
            </CardDescription>
          </div>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create API Key
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {keys.length === 0 ? (
          <Alert>
            <Key className="h-4 w-4" />
            <AlertDescription>
              No API keys created yet. Create your first API key to enable programmatic access.
            </AlertDescription>
          </Alert>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Key Preview</TableHead>
                <TableHead>Permissions</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Used</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => {
                const status = getKeyStatus(key);
                return (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-2 py-1 text-xs">
                        {key.key_prefix}...
                      </code>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {key.permissions.slice(0, 2).map(perm => (
                          <Badge key={perm} variant="outline" className="text-xs">
                            {perm}
                          </Badge>
                        ))}
                        {key.permissions.length > 2 && (
                          <Badge variant="outline" className="text-xs">
                            +{key.permissions.length - 2}
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={status.variant}>{status.label}</Badge>
                    </TableCell>
                    <TableCell>
                      {key.last_used_at
                        ? formatDistanceToNow(new Date(key.last_used_at), { addSuffix: true })
                        : 'Never'}
                    </TableCell>
                    <TableCell>
                      {formatDistanceToNow(new Date(key.created_at), { addSuffix: true })}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRevoke(key.id)}
                        disabled={!!key.revoked_at}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}

        {error && (
          <Alert variant="destructive" className="mt-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {showCreateDialog && (
          <CreateAPIKeyDialog
            organizationId={organizationId}
            onClose={() => setShowCreateDialog(false)}
            onSuccess={(newKey) => {
              fetchAPIKeys();
              setShowCreateDialog(false);
            }}
          />
        )}
      </CardContent>
    </Card>
  );
}

interface CreateAPIKeyDialogProps {
  organizationId?: string;
  onClose: () => void;
  onSuccess: (key: any) => void;
}

function CreateAPIKeyDialog({ organizationId, onClose, onSuccess }: CreateAPIKeyDialogProps) {
  const [name, setName] = useState('');
  const [expiresIn, setExpiresIn] = useState('0');
  const [permissions, setPermissions] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showKey, setShowKey] = useState(true);
  const [copied, setCopied] = useState(false);

  const availablePermissions = [
    { value: 'connections:read', label: 'Read Connections' },
    { value: 'connections:write', label: 'Write Connections' },
    { value: 'queries:read', label: 'Read Queries' },
    { value: 'queries:write', label: 'Execute Queries' },
    { value: 'schemas:read', label: 'Read Schemas' },
    { value: 'templates:read', label: 'Read Templates' },
    { value: 'templates:write', label: 'Write Templates' },
    { value: 'organizations:read', label: 'Read Organization' },
    { value: 'organizations:write', label: 'Manage Organization' },
  ];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (permissions.length === 0) {
      setError('Please select at least one permission');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await fetch('/api/api-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name,
          organization_id: organizationId,
          permissions,
          expires_in_days: parseInt(expiresIn),
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to create API key');
      }

      const data = await response.json();
      setCreatedKey(data.key);
      onSuccess(data.api_key);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create API key');
    } finally {
      setIsSubmitting(false);
    }
  };

  const copyKey = () => {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const togglePermission = (permission: string) => {
    setPermissions(prev =>
      prev.includes(permission)
        ? prev.filter(p => p !== permission)
        : [...prev, permission]
    );
  };

  if (createdKey) {
    return (
      <Dialog open onOpenChange={onClose}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-500" />
              API Key Created Successfully
            </DialogTitle>
            <DialogDescription>
              Make sure to copy your API key now. You won't be able to see it again!
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <Alert>
              <Shield className="h-4 w-4" />
              <AlertDescription>
                Store this key securely. It has the same access as your account.
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label>Your API Key</Label>
              <div className="flex gap-2">
                <div className="relative flex-1">
                  <Input
                    value={showKey ? createdKey : '••••••••••••••••••••••••••••••••'}
                    readOnly
                    className="pr-10 font-mono text-xs"
                  />
                  <Button
                    size="sm"
                    variant="ghost"
                    className="absolute right-1 top-1 h-7 w-7 p-0"
                    onClick={() => setShowKey(!showKey)}
                  >
                    {showKey ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </Button>
                </div>
                <Button size="sm" variant="outline" onClick={copyKey}>
                  {copied ? (
                    <CheckCircle className="h-4 w-4" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                This is the only time you'll see this key. If you lose it, you'll need to create a new one.
              </AlertDescription>
            </Alert>
          </div>

          <DialogFooter>
            <Button onClick={onClose}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open onOpenChange={() => !isSubmitting && onClose()}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Create API Key</DialogTitle>
          <DialogDescription>
            Create a new API key for programmatic access to SQL Studio
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Key Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Production Server"
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="expires">Expiration</Label>
            <Select value={expiresIn} onValueChange={setExpiresIn}>
              <SelectTrigger id="expires">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">Never expire</SelectItem>
                <SelectItem value="7">7 days</SelectItem>
                <SelectItem value="30">30 days</SelectItem>
                <SelectItem value="60">60 days</SelectItem>
                <SelectItem value="90">90 days</SelectItem>
                <SelectItem value="365">1 year</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Permissions</Label>
            <div className="space-y-2 rounded-lg border p-3">
              {availablePermissions.map(perm => (
                <label
                  key={perm.value}
                  className="flex items-center space-x-2 cursor-pointer"
                >
                  <Checkbox
                    checked={permissions.includes(perm.value)}
                    onCheckedChange={() => togglePermission(perm.value)}
                  />
                  <span className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                    {perm.label}
                  </span>
                </label>
              ))}
            </div>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || !name}>
              {isSubmitting ? 'Creating...' : 'Create API Key'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}