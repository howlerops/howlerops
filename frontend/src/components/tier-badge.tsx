/**
 * Tier Badge Component
 *
 * Visual indicator for the current Howlerops tier.
 * Displays tier name with appropriate styling and icon.
 *
 * Features:
 * - Multiple variants (header, inline, card)
 * - Tier-specific colors and icons
 * - Optional expiration warnings
 * - Click handler for tier management
 */

import React, { useMemo } from "react";
import { HardDrive, Cloud, Users } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useTierStore } from "@/store/tier-store";
import { TIER_METADATA } from "@/config/tier-limits";
import type { TierLevel } from "@/types/tiers";
import { cn } from "@/lib/utils";

/**
 * Badge variant types
 */
export type TierBadgeVariant = "header" | "inline" | "card" | "minimal";

/**
 * Tier badge component props
 */
export interface TierBadgeProps {
  /**
   * Display variant
   * - header: Small badge for app header/toolbar
   * - inline: Inline badge for text content
   * - card: Large card-style badge with details
   * - minimal: Just the tier name, no icon
   * @default 'inline'
   */
  variant?: TierBadgeVariant;

  /**
   * Override tier (useful for comparison/marketing)
   * If not provided, uses current tier from store
   */
  tier?: TierLevel;

  /**
   * Show expiration warning if license is expiring soon
   * @default true
   */
  showExpiration?: boolean;

  /**
   * Click handler for tier management
   */
  onClick?: () => void;

  /**
   * Additional CSS classes
   */
  className?: string;

  /**
   * Show development mode indicator
   * @default true
   */
  showDevMode?: boolean;
}

/**
 * Get tier icon component
 */
function getTierIcon(tier: TierLevel): React.ReactNode {
  const iconProps = { className: "w-3.5 h-3.5" };

  switch (tier) {
    case "local":
      return <HardDrive {...iconProps} />;
    case "individual":
      return <Cloud {...iconProps} />;
    case "team":
      return <Users {...iconProps} />;
  }
}

/**
 * Get tier color classes
 */
function getTierColorClasses(tier: TierLevel): {
  badge: string;
  text: string;
  background: string;
} {
  switch (tier) {
    case "local":
      return {
        badge:
          "bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-100 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600",
        text: "text-gray-700 dark:text-gray-100",
        background: "bg-gray-50 dark:bg-gray-900",
      };
    case "individual":
      return {
        badge:
          "bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800 hover:bg-blue-200 dark:hover:bg-blue-800",
        text: "text-blue-700 dark:text-blue-300",
        background: "bg-blue-50 dark:bg-blue-950",
      };
    case "team":
      return {
        badge:
          "bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 border-purple-200 dark:border-purple-800 hover:bg-purple-200 dark:hover:bg-purple-800",
        text: "text-purple-700 dark:text-purple-300",
        background: "bg-purple-50 dark:bg-purple-950",
      };
  }
}

/**
 * Check if license is expiring soon (within 30 days)
 */
function isExpiringSoon(expiresAt?: Date): boolean {
  if (!expiresAt) return false;

  const now = new Date();
  const expiry = new Date(expiresAt);
  const daysUntilExpiry = Math.floor(
    (expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  );

  return daysUntilExpiry <= 30 && daysUntilExpiry > 0;
}

/**
 * Format expiration date
 */
function formatExpiration(expiresAt: Date): string {
  const now = new Date();
  const expiry = new Date(expiresAt);
  const daysUntilExpiry = Math.floor(
    (expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  );

  if (daysUntilExpiry < 0) {
    return "Expired";
  } else if (daysUntilExpiry === 0) {
    return "Expires today";
  } else if (daysUntilExpiry === 1) {
    return "Expires tomorrow";
  } else if (daysUntilExpiry <= 30) {
    return `Expires in ${daysUntilExpiry} days`;
  } else {
    return `Expires ${expiry.toLocaleDateString()}`;
  }
}

/**
 * Tier Badge Component
 *
 * @example
 * ```typescript
 * // Header badge
 * <TierBadge variant="header" onClick={openTierSettings} />
 *
 * // Inline badge
 * <p>Your current plan: <TierBadge variant="inline" /></p>
 *
 * // Card variant with details
 * <TierBadge variant="card" showExpiration />
 *
 * // Comparison badge
 * <TierBadge tier="team" variant="inline" />
 * ```
 */
export function TierBadge({
  variant = "inline",
  tier: overrideTier,
  showExpiration = true,
  onClick,
  className,
  showDevMode = true,
}: TierBadgeProps) {
  const { currentTier, expiresAt, devMode } = useTierStore();

  const tier = overrideTier ?? currentTier;
  const metadata = TIER_METADATA[tier];
  const colors = getTierColorClasses(tier);
  const icon = getTierIcon(tier);

  const expirationWarning = useMemo(() => {
    if (!showExpiration || !expiresAt) return null;
    if (isExpiringSoon(expiresAt)) {
      return formatExpiration(expiresAt);
    }
    return null;
  }, [showExpiration, expiresAt]);

  // Minimal variant - just text
  if (variant === "minimal") {
    return (
      <span
        className={cn(
          "text-sm font-medium",
          colors.text,
          onClick && "cursor-pointer",
          className
        )}
        onClick={onClick}
      >
        {metadata.name}
      </span>
    );
  }

  // Header variant - compact badge
  if (variant === "header") {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        <Badge
          variant="outline"
          className={cn(
            "flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium border transition-colors",
            colors.badge,
            onClick && "cursor-pointer",
            className
          )}
          onClick={onClick}
        >
          {icon}
          <span>{metadata.name}</span>
        </Badge>
        {showDevMode && devMode && (
          <Badge
            variant="outline"
            className="px-2 py-0.5 text-xs bg-yellow-100 text-yellow-700"
          >
            DEV
          </Badge>
        )}
      </div>
    );
  }

  // Inline variant - small badge
  if (variant === "inline") {
    return (
      <Badge
        variant="outline"
        className={cn(
          "inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium border transition-colors",
          colors.badge,
          onClick && "cursor-pointer",
          className
        )}
        onClick={onClick}
      >
        {icon}
        <span>{metadata.name}</span>
        {expirationWarning && (
          <span className="ml-1 text-orange-600 font-semibold">⚠</span>
        )}
      </Badge>
    );
  }

  // Card variant - detailed display
  if (variant === "card") {
    return (
      <div
        className={cn(
          "rounded-lg border p-4 transition-colors",
          colors.background,
          onClick && "cursor-pointer hover:shadow-md",
          className
        )}
        onClick={onClick}
      >
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                "p-2 rounded-lg",
                tier === "local" && "bg-gray-200 dark:bg-gray-700",
                tier === "individual" && "bg-blue-200 dark:bg-blue-800",
                tier === "team" && "bg-purple-200 dark:bg-purple-800"
              )}
            >
              <span
                className={cn(
                  "inline-flex w-5 h-5",
                  tier === "local" && "text-gray-700 dark:text-gray-100",
                  tier === "individual" && "text-blue-700 dark:text-blue-300",
                  tier === "team" && "text-purple-700 dark:text-purple-300"
                )}
              >
                {React.isValidElement(icon) ? icon : null}
              </span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className={cn("text-lg font-semibold", colors.text)}>
                  {metadata.name}
                </h3>
                {showDevMode && devMode && (
                  <Badge
                    variant="outline"
                    className="text-xs bg-yellow-100 text-yellow-700"
                  >
                    DEV MODE
                  </Badge>
                )}
              </div>
              <p className="text-sm text-muted-foreground mt-0.5">
                {metadata.description}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className={cn("text-lg font-bold", colors.text)}>
              {metadata.priceLabel}
            </p>
          </div>
        </div>

        {expirationWarning && (
          <div className="mt-3 px-3 py-2 rounded-md bg-orange-50 border border-orange-200">
            <p className="text-sm text-orange-700 font-medium">
              ⚠ {expirationWarning}
            </p>
          </div>
        )}
      </div>
    );
  }

  return null;
}

/**
 * Tier Badge List Component
 * Displays all available tiers for comparison
 *
 * @example
 * ```typescript
 * <TierBadgeList
 *   onSelect={(tier) => console.log('Selected:', tier)}
 *   highlightCurrent
 * />
 * ```
 */
export interface TierBadgeListProps {
  /**
   * Selection handler
   */
  onSelect?: (tier: TierLevel) => void;

  /**
   * Highlight the current tier
   * @default true
   */
  highlightCurrent?: boolean;

  /**
   * Display variant for each badge
   * @default 'card'
   */
  variant?: TierBadgeVariant;

  /**
   * Additional CSS classes
   */
  className?: string;
}

export function TierBadgeList({
  onSelect,
  highlightCurrent = true,
  variant = "card",
  className,
}: TierBadgeListProps) {
  const { currentTier } = useTierStore();

  const tiers: TierLevel[] = ["local", "individual", "team"];

  return (
    <div
      className={cn(
        "grid gap-4",
        variant === "card" && "md:grid-cols-3",
        className
      )}
    >
      {tiers.map((tier) => (
        <div
          key={tier}
          className={cn(
            "relative",
            highlightCurrent && tier === currentTier && "ring-2 ring-offset-2",
            tier === "local" && "ring-gray-400",
            tier === "individual" && "ring-blue-400",
            tier === "team" && "ring-purple-400"
          )}
        >
          {highlightCurrent && tier === currentTier && (
            <div className="absolute -top-2 -right-2 z-10">
              <Badge className="bg-green-500 text-white text-xs">Current</Badge>
            </div>
          )}
          <TierBadge
            tier={tier}
            variant={variant}
            onClick={() => onSelect?.(tier)}
          />
        </div>
      ))}
    </div>
  );
}
