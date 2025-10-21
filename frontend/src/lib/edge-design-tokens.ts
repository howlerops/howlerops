/**
 * Comprehensive design token system for database schema relationship edges
 * Based on industry best practices and WCAG 2.1 AA accessibility standards
 */

export const edgeDesignSystem = {
  colors: {
    standard: {
      hasOne: {
        default: '#3b82f6',
        hover: '#2563eb',
        selected: '#1d4ed8',
        dimmed: '#93c5fd',
      },
      hasMany: {
        default: '#f59e0b',
        hover: '#d97706',
        selected: '#b45309',
        dimmed: '#fcd34d',
      },
      belongsTo: {
        default: '#8b5cf6',
        hover: '#7c3aed',
        selected: '#6d28d9',
        dimmed: '#c4b5fd',
      },
      manyToMany: {
        default: '#ef4444',
        hover: '#dc2626',
        selected: '#b91c1c',
        dimmed: '#fca5a5',
      },
    },
    crossSchema: {
      hasOne: '#06b6d4',
      hasMany: '#0ea5e9',
      belongsTo: '#6366f1',
      manyToMany: '#ec4899',
    },
    colorBlind: {
      deuteranopia: {
        hasOne: '#3b82f6',
        hasMany: '#f59e0b',
        belongsTo: '#8b5cf6',
        manyToMany: '#ec4899',
      },
      protanopia: {
        hasOne: '#06b6d4',
        hasMany: '#eab308',
        belongsTo: '#8b5cf6',
        manyToMany: '#0284c7',
      },
      tritanopia: {
        hasOne: '#ec4899',
        hasMany: '#ef4444',
        belongsTo: '#0ea5e9',
        manyToMany: '#7c3aed',
      },
    },
  },

  patterns: {
    solid: '0',
    dashed: '8 4',
    dotted: '2 3',
    dashDot: '8 4 2 4',
    longDash: '12 6',
    doubleDash: '4 2 4 6',
  },

  widths: {
    default: 2,
    hover: 3,
    selected: 3.5,
    highlighted: 2.5,
    dimmed: 1.5,
  },

  opacity: {
    default: 0.85,
    hover: 1.0,
    selected: 1.0,
    highlighted: 0.95,
    dimmed: 0.25,
    hidden: 0.1,
  },

  animations: {
    flow: {
      duration: '3s',
      easing: 'linear',
      iterations: 'infinite' as const,
    },
    pulse: {
      duration: '2s',
      easing: 'ease-in-out',
      iterations: 'infinite' as const,
    },
    transition: {
      duration: '0.15s',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },

  labels: {
    typography: {
      fontSize: '11px',
      fontWeight: '600',
      fontFamily: 'Inter, system-ui, sans-serif',
      letterSpacing: '0.02em',
    },
    container: {
      background: 'rgba(255, 255, 255, 0.95)',
      backgroundDark: 'rgba(15, 23, 42, 0.95)',
      border: '1px solid rgba(0, 0, 0, 0.08)',
      borderDark: '1px solid rgba(255, 255, 255, 0.12)',
      borderRadius: '4px',
      padding: '2px 6px',
      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
      backdropFilter: 'blur(8px)',
    },
    visibility: {
      hideBelow: 0.75,
      cardinalityOnly: [0.75, 1.0] as const,
      fullDetail: 1.0,
    },
  },

  markers: {
    hasOne: { type: 'arrowclosed' as const, width: 12, height: 12 },
    hasMany: { type: 'arrowclosed' as const, width: 16, height: 12 },
    belongsTo: { type: 'arrowclosed' as const, width: 10, height: 10 },
    manyToMany: { type: 'arrowclosed' as const, width: 16, height: 12 },
  },

  crossSchema: {
    indicator: '↗',
    doubleStroke: {
      outerWidth: 5,
      outerOpacity: 0.2,
    },
    glow: {
      blur: 3,
      color: 'rgba(99, 102, 241, 0.4)',
    },
  },

  density: {
    bundleOffset: 20,
    bundleWidthMultiplier: 0.85,
    curvatureIncrement: 0.1,
    bridge: {
      gap: 6,
      opacity: 0.8,
    },
  },

  zoom: {
    minimal: { max: 0.5, strokeWidth: 1, labels: false },
    reduced: { min: 0.5, max: 0.75, strokeWidth: 1.5, labels: false },
    standard: { min: 0.75, max: 1.0, strokeWidth: 2, cardinality: true },
    detailed: { min: 1.0, strokeWidth: 2, fullLabels: true },
    enhanced: { min: 1.5, strokeWidth: 2.5, constraints: true },
  },
} as const

/**
 * Get cardinality symbol for relationship type
 */
export function getCardinalitySymbol(relation: 'hasOne' | 'hasMany' | 'belongsTo' | 'manyToMany'): string {
  const symbols = {
    hasOne: '1—1',
    hasMany: '1—∞',
    belongsTo: '∞—1',
    manyToMany: '∞—∞',
  }
  return symbols[relation]
}

/**
 * Get pattern for relationship type
 */
export function getRelationshipPattern(
  relation: 'hasOne' | 'hasMany' | 'belongsTo' | 'manyToMany',
  crossSchema: boolean = false
): string {
  if (crossSchema) {
    return relation === 'belongsTo' ? edgeDesignSystem.patterns.doubleDash : edgeDesignSystem.patterns.longDash
  }

  const patterns = {
    hasOne: edgeDesignSystem.patterns.solid,
    hasMany: edgeDesignSystem.patterns.solid,
    belongsTo: edgeDesignSystem.patterns.dashed,
    manyToMany: edgeDesignSystem.patterns.dashDot,
  }
  return patterns[relation]
}

/**
 * Get color for relationship type and state
 */
export function getEdgeColor(
  relation: 'hasOne' | 'hasMany' | 'belongsTo' | 'manyToMany',
  state: 'default' | 'hover' | 'selected' | 'dimmed' = 'default',
  crossSchema: boolean = false
): string {
  if (crossSchema) {
    return edgeDesignSystem.colors.crossSchema[relation]
  }

  return edgeDesignSystem.colors.standard[relation][state]
}
