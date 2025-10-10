/**
 * useConnectionStatus Hook - Monitors WebSocket connection health and status
 * Provides detailed connection information and health metrics
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useWebSocket } from './useWebSocket';
import { ConnectionStatus } from '../../types/websocket';

interface ConnectionMetrics {
  latency: number | null;
  lastPingTime: number | null;
  reconnectAttempts: number;
  totalReconnects: number;
  uptime: number;
  downtime: number;
  lastConnectedAt: number | null;
  lastDisconnectedAt: number | null;
  connectionStability: number; // 0-1 score
  messagesSent: number;
  messagesReceived: number;
  bytesTransferred: number;
}

interface ConnectionHealth {
  overall: 'healthy' | 'degraded' | 'unhealthy';
  score: number; // 0-100
  issues: string[];
  recommendations: string[];
}

interface NetworkQuality {
  level: 'excellent' | 'good' | 'fair' | 'poor';
  latency: number | null;
  stability: number;
  description: string;
}

const initialMetrics: ConnectionMetrics = {
  latency: null,
  lastPingTime: null,
  reconnectAttempts: 0,
  totalReconnects: 0,
  uptime: 0,
  downtime: 0,
  lastConnectedAt: null,
  lastDisconnectedAt: null,
  connectionStability: 1,
  messagesSent: 0,
  messagesReceived: 0,
  bytesTransferred: 0,
};

export function useConnectionStatus() {
  const { connectionState, socket, healthCheck, getStats } = useWebSocket();

  // State
  const [metrics, setMetrics] = useState<ConnectionMetrics>(initialMetrics);
  const [health, setHealth] = useState<ConnectionHealth>({
    overall: 'healthy',
    score: 100,
    issues: [],
    recommendations: [],
  });
  const [networkQuality, setNetworkQuality] = useState<NetworkQuality>({
    level: 'excellent',
    latency: null,
    stability: 1,
    description: 'Connection quality is excellent',
  });

  // Refs
  const metricsIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const pingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const connectionStartRef = useRef<number | null>(null);
  const stabilityHistoryRef = useRef<boolean[]>([]);

  /**
   * Calculate connection stability
   */
  const calculateStability = useCallback(() => {
    const history = stabilityHistoryRef.current;
    if (history.length === 0) return 1;

    const successfulConnections = history.filter(connected => connected).length;
    return successfulConnections / history.length;
  }, []);

  /**
   * Update connection stability history
   */
  const updateStabilityHistory = useCallback((connected: boolean) => {
    stabilityHistoryRef.current.push(connected);

    // Keep only last 100 data points
    if (stabilityHistoryRef.current.length > 100) {
      stabilityHistoryRef.current = stabilityHistoryRef.current.slice(-100);
    }

    const stability = calculateStability();
    setMetrics(prev => ({ ...prev, connectionStability: stability }));
  }, [calculateStability]);

  /**
   * Measure latency
   */
  const measureLatency = useCallback(async (): Promise<number | null> => {
    if (!socket?.connected) return null;

    try {
      const startTime = performance.now();

      const latency = await new Promise<number>((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('Ping timeout')), 5000);

        socket.emit('ping');
        socket.once('pong', () => {
          clearTimeout(timeout);
          const endTime = performance.now();
          resolve(endTime - startTime);
        });
      });

      setMetrics(prev => ({
        ...prev,
        latency,
        lastPingTime: Date.now(),
      }));

      return latency;

    } catch (error) {
      console.warn('Latency measurement failed:', error);
      return null;
    }
  }, [socket]);

  /**
   * Assess network quality
   */
  const assessNetworkQuality = useCallback((latency: number | null, stability: number) => {
    let level: NetworkQuality['level'];
    let description: string;

    if (latency === null || latency > 1000 || stability < 0.5) {
      level = 'poor';
      description = 'Connection quality is poor. Consider checking your network.';
    } else if (latency > 500 || stability < 0.8) {
      level = 'fair';
      description = 'Connection quality is fair. Some operations may be slower.';
    } else if (latency > 200 || stability < 0.95) {
      level = 'good';
      description = 'Connection quality is good. Most operations should work well.';
    } else {
      level = 'excellent';
      description = 'Connection quality is excellent. All operations should be fast and reliable.';
    }

    setNetworkQuality({
      level,
      latency,
      stability,
      description,
    });
  }, []);

  /**
   * Calculate health score and issues
   */
  const calculateHealth = useCallback((metrics: ConnectionMetrics, status: ConnectionStatus): ConnectionHealth => {
    let score = 100;
    const issues: string[] = [];
    const recommendations: string[] = [];

    // Connection status
    if (status === 'disconnected') {
      score -= 50;
      issues.push('Not connected to server');
      recommendations.push('Check your network connection');
    } else if (status === 'reconnecting') {
      score -= 30;
      issues.push('Attempting to reconnect');
    } else if (status === 'error') {
      score -= 60;
      issues.push('Connection error occurred');
      recommendations.push('Refresh the page or check server status');
    }

    // Latency
    if (metrics.latency !== null) {
      if (metrics.latency > 1000) {
        score -= 20;
        issues.push('High latency detected');
        recommendations.push('Check your network speed');
      } else if (metrics.latency > 500) {
        score -= 10;
        issues.push('Elevated latency');
      }
    }

    // Stability
    if (metrics.connectionStability < 0.8) {
      score -= 15;
      issues.push('Connection instability detected');
      recommendations.push('Check for network interference');
    }

    // Reconnect attempts
    if (metrics.reconnectAttempts > 3) {
      score -= 15;
      issues.push('Multiple reconnection attempts');
      recommendations.push('Consider refreshing the page');
    }

    // Total reconnects
    if (metrics.totalReconnects > 5) {
      score -= 10;
      issues.push('Frequent reconnections');
    }

    // Determine overall health
    let overall: ConnectionHealth['overall'];
    if (score >= 80) {
      overall = 'healthy';
    } else if (score >= 50) {
      overall = 'degraded';
    } else {
      overall = 'unhealthy';
    }

    return {
      overall,
      score: Math.max(0, score),
      issues,
      recommendations,
    };
  }, []);

  /**
   * Update metrics based on connection state
   */
  const updateMetrics = useCallback(() => {
    const now = Date.now();

    setMetrics(prev => {
      const newMetrics = { ...prev };

      // Update uptime/downtime
      if (connectionState.status === 'connected') {
        if (connectionStartRef.current) {
          newMetrics.uptime = now - connectionStartRef.current;
        }
      } else {
        if (prev.lastDisconnectedAt) {
          newMetrics.downtime = now - prev.lastDisconnectedAt;
        }
      }

      // Update reconnect attempts
      newMetrics.reconnectAttempts = connectionState.reconnectAttempts;

      return newMetrics;
    });

    // Assess network quality
    assessNetworkQuality(metrics.latency, metrics.connectionStability);

    // Calculate health
    const newHealth = calculateHealth(metrics, connectionState.status);
    setHealth(newHealth);
  }, [connectionState, metrics, assessNetworkQuality, calculateHealth]);

  /**
   * Handle connection events
   */
  useEffect(() => {
    const now = Date.now();

    if (connectionState.status === 'connected') {
      if (!connectionStartRef.current) {
        connectionStartRef.current = now;
      }

      setMetrics(prev => ({
        ...prev,
        lastConnectedAt: now,
      }));

      updateStabilityHistory(true);

      // If this was a reconnection
      if (metrics.reconnectAttempts > 0) {
        setMetrics(prev => ({
          ...prev,
          totalReconnects: prev.totalReconnects + 1,
          reconnectAttempts: 0,
        }));
      }

    } else if (connectionState.status === 'disconnected') {
      connectionStartRef.current = null;

      setMetrics(prev => ({
        ...prev,
        lastDisconnectedAt: now,
      }));

      updateStabilityHistory(false);
    }
  }, [connectionState.status, metrics.reconnectAttempts, updateStabilityHistory]);

  /**
   * Start periodic monitoring
   */
  useEffect(() => {
    // Metrics update interval
    metricsIntervalRef.current = setInterval(() => {
      updateMetrics();
    }, 5000); // Every 5 seconds

    // Ping interval for latency measurement
    pingIntervalRef.current = setInterval(() => {
      if (connectionState.status === 'connected') {
        measureLatency();
      }
    }, 10000); // Every 10 seconds

    return () => {
      if (metricsIntervalRef.current) {
        clearInterval(metricsIntervalRef.current);
      }
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current);
      }
    };
  }, [connectionState.status, updateMetrics, measureLatency]);

  /**
   * Force health check
   */
  const performHealthCheck = useCallback(async () => {
    const isHealthy = await healthCheck();

    setHealth(prev => ({
      ...prev,
      overall: isHealthy ? 'healthy' : 'unhealthy',
      score: isHealthy ? Math.max(prev.score, 80) : Math.min(prev.score, 40),
    }));

    return isHealthy;
  }, [healthCheck]);

  /**
   * Reset metrics
   */
  const resetMetrics = useCallback(() => {
    setMetrics(initialMetrics);
    stabilityHistoryRef.current = [];
    connectionStartRef.current = null;
  }, []);

  /**
   * Get detailed status report
   */
  const getStatusReport = useCallback(() => {
    const wsStats = getStats();

    return {
      connection: {
        status: connectionState.status,
        serverInfo: connectionState.serverInfo,
        lastConnected: connectionState.lastConnected,
        error: connectionState.error,
      },
      metrics,
      health,
      networkQuality,
      websocket: wsStats,
      timestamp: Date.now(),
    };
  }, [connectionState, metrics, health, networkQuality, getStats]);

  /**
   * Format uptime/downtime for display
   */
  const formatDuration = useCallback((ms: number): string => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${Math.round(ms / 1000)}s`;
    if (ms < 3600000) return `${Math.round(ms / 60000)}m`;
    return `${Math.round(ms / 3600000)}h`;
  }, []);

  return {
    // Current state
    status: connectionState.status,
    metrics,
    health,
    networkQuality,

    // Actions
    performHealthCheck,
    measureLatency,
    resetMetrics,

    // Data access
    getStatusReport,
    formatDuration,

    // Computed values
    isHealthy: health.overall === 'healthy',
    isDegraded: health.overall === 'degraded',
    isUnhealthy: health.overall === 'unhealthy',
    hasIssues: health.issues.length > 0,
    hasRecommendations: health.recommendations.length > 0,
  };
}