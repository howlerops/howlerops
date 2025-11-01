import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Plus, Trash2, AlertCircle, Globe, Network } from 'lucide-react';
import { Badge } from '@/components/ui/badge';

interface IPWhitelistEntry {
  id: string;
  ip_address?: string;
  ip_range?: string;
  description?: string;
  created_by: string;
  created_at: string;
}

interface IPWhitelistManagerProps {
  organizationId: string;
}

export function IPWhitelistManager({ organizationId }: IPWhitelistManagerProps) {
  const [entries, setEntries] = useState<IPWhitelistEntry[]>([]);
  const [_isLoading, setIsLoading] = useState(true);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchWhitelist = useCallback(async () => {
    try {
      const response = await fetch(`/api/organizations/${organizationId}/ip-whitelist`);
      if (!response.ok) throw new Error('Failed to fetch IP whitelist');
      const data = await response.json();
      setEntries(data);
    } catch {
      setError('Failed to load IP whitelist');
    } finally {
      setIsLoading(false);
    }
  }, [organizationId]);

  useEffect(() => {
    fetchWhitelist();
  }, [fetchWhitelist]);

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to remove this IP from the whitelist?')) {
      return;
    }

    try {
      const response = await fetch(
        `/api/organizations/${organizationId}/ip-whitelist/${id}`,
        {
          method: 'DELETE',
        }
      );

      if (!response.ok) throw new Error('Failed to remove IP');

      setEntries(entries.filter(e => e.id !== id));
    } catch {
      setError('Failed to remove IP from whitelist');
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Network className="h-5 w-5" />
              IP Whitelist
            </CardTitle>
            <CardDescription>
              Control which IP addresses can access your organization
            </CardDescription>
          </div>
          <Button onClick={() => setShowAddDialog(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add IP
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {entries.length === 0 ? (
          <Alert>
            <Globe className="h-4 w-4" />
            <AlertDescription>
              No IP restrictions configured. All IP addresses can access your organization.
            </AlertDescription>
          </Alert>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IP Address / Range</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Description</TableHead>
                <TableHead>Added By</TableHead>
                <TableHead>Added On</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="font-mono">
                    {entry.ip_address || entry.ip_range}
                  </TableCell>
                  <TableCell>
                    <Badge variant={entry.ip_range ? 'secondary' : 'default'}>
                      {entry.ip_range ? 'CIDR Range' : 'Single IP'}
                    </Badge>
                  </TableCell>
                  <TableCell>{entry.description || '-'}</TableCell>
                  <TableCell>{entry.created_by}</TableCell>
                  <TableCell>
                    {new Date(entry.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDelete(entry.id)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {error && (
          <Alert variant="destructive" className="mt-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {showAddDialog && (
          <AddIPDialog
            organizationId={organizationId}
            onClose={() => setShowAddDialog(false)}
            onSuccess={(entry) => {
              setEntries([...entries, entry]);
              setShowAddDialog(false);
            }}
          />
        )}
      </CardContent>
    </Card>
  );
}

interface AddIPDialogProps {
  organizationId: string;
  onClose: () => void;
  onSuccess: (entry: IPWhitelistEntry) => void;
}

function AddIPDialog({ organizationId, onClose, onSuccess }: AddIPDialogProps) {
  const [ipType, setIpType] = useState<'single' | 'range'>('single');
  const [ipAddress, setIpAddress] = useState('');
  const [ipRange, setIpRange] = useState('');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      const body: { description: string; ip_address?: string; ip_range?: string } = { description };
      if (ipType === 'single') {
        body.ip_address = ipAddress;
      } else {
        body.ip_range = ipRange;
      }

      const response = await fetch(`/api/organizations/${organizationId}/ip-whitelist`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        const data = await response.text();
        throw new Error(data || 'Failed to add IP to whitelist');
      }

      const entry = await response.json();
      onSuccess(entry);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add IP');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open onOpenChange={() => !isSubmitting && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add IP to Whitelist</DialogTitle>
          <DialogDescription>
            Add a single IP address or CIDR range to your organization's whitelist
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label>IP Type</Label>
            <div className="flex gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  checked={ipType === 'single'}
                  onChange={() => setIpType('single')}
                />
                Single IP Address
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  checked={ipType === 'range'}
                  onChange={() => setIpType('range')}
                />
                CIDR Range
              </label>
            </div>
          </div>

          {ipType === 'single' ? (
            <div className="space-y-2">
              <Label htmlFor="ip-address">IP Address</Label>
              <Input
                id="ip-address"
                type="text"
                value={ipAddress}
                onChange={(e) => setIpAddress(e.target.value)}
                placeholder="192.168.1.1"
                pattern="^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$"
                required
              />
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="ip-range">CIDR Range</Label>
              <Input
                id="ip-range"
                type="text"
                value={ipRange}
                onChange={(e) => setIpRange(e.target.value)}
                placeholder="192.168.1.0/24"
                pattern="^(?:[0-9]{1,3}\.){3}[0-9]{1,3}\/[0-9]{1,2}$"
                required
              />
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="description">Description (Optional)</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="e.g., Office network, VPN server"
              rows={2}
            />
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
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Adding...' : 'Add to Whitelist'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}