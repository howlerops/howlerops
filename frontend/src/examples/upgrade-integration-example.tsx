/**
 * Upgrade System Integration Example
 *
 * This file demonstrates how to integrate the upgrade prompt system
 * into your Howlerops components.
 */

import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ConnectionLimitIndicator,
  QueryHistoryIndicator,
} from "@/components/value-indicators";
import { showSoftLimitToast } from "@/components/soft-limits";
import { UsageStats } from "@/components/usage-stats";
import { useUpgrade } from "@/components/upgrade-provider";
import { useTierStore } from "@/store/tier-store";
import { useConnectionStore } from "@/store/connection-store";
import { trackQueryExecution } from "@/lib/upgrade-reminders";

/**
 * Example 1: Connection Management with Soft Limits
 */
export function ConnectionManagementExample() {
  const { connections } = useConnectionStore();
  const { checkLimit } = useTierStore();
  const { showUpgrade } = useUpgrade();

  const handleAddConnection = () => {
    const limitCheck = checkLimit("connections", connections.length + 1);

    if (!limitCheck.allowed) {
      // Show soft limit toast
      showSoftLimitToast({
        limitType: "connections",
        usage: connections.length,
        softLimit: limitCheck.limit || 5,
        onUpgrade: () => showUpgrade("connections"),
      });
      return;
    }

    // Add connection...
    console.log("Adding connection...");
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Database Connections</CardTitle>
          <ConnectionLimitIndicator variant="badge" />
        </div>
      </CardHeader>
      <CardContent>
        <Button onClick={handleAddConnection}>Add Connection</Button>
      </CardContent>
    </Card>
  );
}

/**
 * Example 2: Query History with Usage Indicator
 */
export function QueryHistoryExample() {
  const [queries, setQueries] = useState<string[]>([]);

  const _handleExecuteQuery = (query: string) => {
    // Execute query...
    console.log("Executing:", query);

    // Track for activity monitoring
    trackQueryExecution();

    // Add to history
    setQueries([...queries, query]);
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Query History</CardTitle>
          <QueryHistoryIndicator variant="inline" showUpgradeCTA />
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Banner shown when approaching limit */}
        <QueryHistoryIndicator variant="banner" showThreshold={80} />

        {/* Query list */}
        <div className="space-y-2">
          {queries.map((q, idx) => (
            <div key={idx} className="p-2 bg-muted rounded">
              {q}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * Example 3: Settings Panel with Usage Stats
 */
export function SettingsExample() {
  const { currentTier } = useTierStore();
  const { showUpgrade } = useUpgrade();

  return (
    <div className="space-y-6 max-w-4xl">
      <Card>
        <CardHeader>
          <CardTitle>Account</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <p className="text-sm text-muted-foreground">Current Plan</p>
              <p className="text-lg font-semibold capitalize">{currentTier}</p>
            </div>
            {currentTier === "local" && (
              <Button
                className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
                onClick={() => showUpgrade("manual")}
              >
                Upgrade to Pro
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      <UsageStats showUpgradeCTA />
    </div>
  );
}

/**
 * Example 4: Locked Feature with Upgrade Prompt
 */
export function LockedFeatureExample() {
  const { hasFeature } = useTierStore();
  const { showUpgrade } = useUpgrade();

  const canUseTeamFeatures = hasFeature("teamSharing");

  if (!canUseTeamFeatures) {
    return (
      <Card
        className="cursor-pointer hover:border-purple-300 transition-colors"
        onClick={() => showUpgrade("feature")}
      >
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Team Collaboration</CardTitle>
            <div className="flex items-center gap-2 text-purple-600">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z"
                  clipRule="evenodd"
                />
              </svg>
              <span className="text-sm font-medium">Team Plan</span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="opacity-60">
          <p className="text-sm text-muted-foreground">
            Share connections and queries with your team. Available in Team
            plan.
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={(e) => {
              e.stopPropagation();
              showUpgrade("feature");
            }}
          >
            Learn More
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Team Collaboration</CardTitle>
      </CardHeader>
      <CardContent>
        {/* Full feature here */}
        <p>Team collaboration features enabled!</p>
      </CardContent>
    </Card>
  );
}

/**
 * Example 5: Export with File Size Check
 */
export function ExportExample() {
  const { showUpgrade } = useUpgrade();

  const handleExport = (dataSize: number) => {
    const limits = useTierStore.getState().getLimits();
    const maxSize = limits.exportFileSize;

    if (dataSize > maxSize) {
      // Show upgrade prompt
      showSoftLimitToast({
        limitType: "export",
        usage: dataSize,
        softLimit: maxSize,
        onUpgrade: () => showUpgrade("export"),
      });
      return;
    }

    // Perform export...
    console.log("Exporting data...");
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Export Data</CardTitle>
      </CardHeader>
      <CardContent>
        <Button onClick={() => handleExport(15 * 1024 * 1024)}>
          Export Large Dataset
        </Button>
      </CardContent>
    </Card>
  );
}

/**
 * Complete Example: App Integration
 */
export function CompleteAppExample() {
  return (
    <div className="p-8 space-y-8">
      <h1 className="text-3xl font-bold">Upgrade System Examples</h1>

      <div className="grid gap-6 md:grid-cols-2">
        <ConnectionManagementExample />
        <QueryHistoryExample />
        <LockedFeatureExample />
        <ExportExample />
      </div>

      <SettingsExample />
    </div>
  );
}
