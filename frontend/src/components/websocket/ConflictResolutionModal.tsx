/**
 * Conflict Resolution Modal - UI for resolving table edit conflicts
 * Provides interactive conflict resolution with multiple strategies
 */

import React, { useState, useCallback, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Textarea } from '../ui/textarea';
import { AlertTriangle, Clock, Users, GitMerge } from 'lucide-react';
import { useConflictResolution } from '../../hooks/websocket';

interface ConflictResolutionModalProps {
  isOpen: boolean;
  onClose: () => void;
  conflictId: string | null;
}

export function ConflictResolutionModal({
  isOpen,
  onClose,
  conflictId,
}: ConflictResolutionModalProps) {
  const {
    activeConflicts,
    strategies,
    resolveConflict,
    getSuggestedResolution,
  } = useConflictResolution();

  const [selectedStrategy, setSelectedStrategy] = useState<string>('');
  const [customValue, setCustomValue] = useState<string>('');
  const [isResolving, setIsResolving] = useState(false);

  const conflict = activeConflicts.find(c => c.id === conflictId);
  const suggestion = conflict ? getSuggestedResolution(conflict.id) : null;

  /**
   * Parse custom value based on original data type
   */
  const parseCustomValue = useCallback((value: string): unknown => {
    if (!conflict) return value;

    const originalType = typeof conflict.localValue;

    try {
      switch (originalType) {
        case 'number':
          return parseFloat(value);
        case 'boolean':
          return value.toLowerCase() === 'true';
        case 'object':
          return JSON.parse(value);
        default:
          return value;
      }
    } catch {
      return value; // Fallback to string
    }
  }, [conflict]);

  /**
   * Handle conflict resolution
   */
  const handleResolve = useCallback(async () => {
    if (!conflict || !selectedStrategy) return;

    setIsResolving(true);

    try {
      const value = selectedStrategy === 'custom' ? parseCustomValue(customValue) : undefined;
      await resolveConflict(conflict.id, selectedStrategy, value);
      onClose();
    } catch (error) {
      console.error('Failed to resolve conflict:', error);
      // Could show error toast here
    } finally {
      setIsResolving(false);
    }
  }, [conflict, selectedStrategy, customValue, resolveConflict, onClose, parseCustomValue]);

  /**
   * Format value for display
   */
  const formatValue = useCallback((value: unknown): string => {
    if (value === null || value === undefined) return 'null';
    if (typeof value === 'object') return JSON.stringify(value, null, 2);
    return String(value);
  }, []);

  /**
   * Get conflict type color
   */
  const getConflictTypeColor = useCallback((type: string) => {
    switch (type) {
      case 'value':
        return 'bg-yellow-100 text-yellow-800';
      case 'type':
        return 'bg-red-100 text-red-800';
      case 'structural':
        return 'bg-purple-100 text-purple-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  }, []);

  /**
   * Get strategy description
   */
  const getStrategyDescription = useCallback((strategyId: string): string => {
    const strategy = strategies.find(s => s.id === strategyId);
    return strategy?.description || '';
  }, [strategies]);

  // Reset form when conflict changes
  useEffect(() => {
    if (conflict && suggestion) {
      setSelectedStrategy(suggestion.strategyId);
      setCustomValue(formatValue(suggestion.value));
    } else {
      setSelectedStrategy('');
      setCustomValue('');
    }
  }, [conflict, suggestion, formatValue]);

  if (!conflict) return null;

  const conflictAge = Date.now() - conflict.timestamp;
  const ageInSeconds = Math.floor(conflictAge / 1000);

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-yellow-500" />
            Resolve Conflict
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-6">
          {/* Conflict Info */}
          <div className="bg-gray-50 p-4 rounded-lg">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <label className="font-medium text-gray-700">Table:</label>
                <div>{conflict.tableId}</div>
              </div>
              <div>
                <label className="font-medium text-gray-700">Column:</label>
                <div>{conflict.column}</div>
              </div>
              <div>
                <label className="font-medium text-gray-700">Row ID:</label>
                <div>{conflict.rowId}</div>
              </div>
              <div className="flex items-center gap-2">
                <label className="font-medium text-gray-700">Type:</label>
                <Badge className={getConflictTypeColor(conflict.conflictType)}>
                  {conflict.conflictType}
                </Badge>
              </div>
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-gray-500" />
                <span className="text-gray-600">
                  {ageInSeconds < 60 ? `${ageInSeconds}s ago` : `${Math.floor(ageInSeconds / 60)}m ago`}
                </span>
              </div>
            </div>
          </div>

          {/* Value Comparison */}
          <div className="grid grid-cols-3 gap-4">
            {/* Local Value */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                <Users className="h-4 w-4" />
                Your Value
              </label>
              <div className="bg-blue-50 border border-blue-200 rounded p-3">
                <pre className="text-sm whitespace-pre-wrap text-blue-800">
                  {formatValue(conflict.localValue)}
                </pre>
              </div>
            </div>

            {/* Remote Value */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                <GitMerge className="h-4 w-4" />
                Remote Value
              </label>
              <div className="bg-green-50 border border-green-200 rounded p-3">
                <pre className="text-sm whitespace-pre-wrap text-green-800">
                  {formatValue(conflict.remoteValue)}
                </pre>
              </div>
            </div>

            {/* Base Value */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">
                Original Value
              </label>
              <div className="bg-gray-50 border border-gray-200 rounded p-3">
                <pre className="text-sm whitespace-pre-wrap text-gray-600">
                  {formatValue(conflict.baseValue)}
                </pre>
              </div>
            </div>
          </div>

          {/* Resolution Strategy */}
          <div className="space-y-4">
            <label className="text-sm font-medium text-gray-700">
              Resolution Strategy
            </label>

            <Select value={selectedStrategy} onValueChange={setSelectedStrategy}>
              <SelectTrigger>
                <SelectValue placeholder="Choose how to resolve this conflict" />
              </SelectTrigger>
              <SelectContent>
                {strategies.map(strategy => (
                  <SelectItem key={strategy.id} value={strategy.id}>
                    <div className="flex flex-col">
                      <span>{strategy.name}</span>
                      <span className="text-xs text-gray-500">
                        {strategy.description}
                      </span>
                    </div>
                  </SelectItem>
                ))}
                <SelectItem value="custom">
                  <div className="flex flex-col">
                    <span>Custom Value</span>
                    <span className="text-xs text-gray-500">
                      Enter a custom resolution value
                    </span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>

            {selectedStrategy && (
              <div className="text-sm text-gray-600 bg-blue-50 p-3 rounded">
                {getStrategyDescription(selectedStrategy)}
              </div>
            )}

            {/* Suggestion */}
            {suggestion && suggestion.strategyId === selectedStrategy && (
              <div className="bg-yellow-50 border border-yellow-200 rounded p-3">
                <div className="flex items-center gap-2 text-sm text-yellow-800">
                  <AlertTriangle className="h-4 w-4" />
                  <span>
                    Suggested resolution (confidence: {Math.round(suggestion.confidence * 100)}%)
                  </span>
                </div>
              </div>
            )}
          </div>

          {/* Custom Value Input */}
          {selectedStrategy === 'custom' && (
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">
                Custom Value
              </label>
              <Textarea
                value={customValue}
                onChange={(e) => setCustomValue(e.target.value)}
                placeholder="Enter the resolved value..."
                rows={4}
              />
              <div className="text-xs text-gray-500">
                Enter the value as it should appear. For objects, use JSON format.
              </div>
            </div>
          )}

          {/* Preview */}
          {selectedStrategy && (
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">
                Preview Resolution
              </label>
              <div className="bg-green-50 border border-green-200 rounded p-3">
                <pre className="text-sm whitespace-pre-wrap text-green-800">
                  {selectedStrategy === 'custom'
                    ? customValue
                    : formatValue(
                        selectedStrategy === 'last_write_wins'
                          ? conflict.remoteValue
                          : selectedStrategy === 'first_write_wins'
                          ? conflict.localValue
                          : suggestion?.value || 'N/A'
                      )
                  }
                </pre>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isResolving}>
            Cancel
          </Button>
          <Button
            onClick={handleResolve}
            disabled={!selectedStrategy || isResolving}
            className="min-w-[100px]"
          >
            {isResolving ? 'Resolving...' : 'Resolve Conflict'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}