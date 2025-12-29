import React, { memo, useCallback, useMemo, useState } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  EdgeProps,
  getSmoothStepPath,
} from 'reactflow'

import { edgeDesignSystem, getCardinalitySymbol } from '@/lib/edge-design-tokens'
import { EdgeConfig } from '@/types/schema-visualizer'

interface CustomEdgeData {
  data?: EdgeConfig
  onEdgeHover?: (edgeId: string | null) => void
  isHighlighted?: boolean
  isDimmed?: boolean
}

// Pre-computed base styles to avoid object recreation
const baseEdgeStyles = {
  default: {
    opacity: edgeDesignSystem.opacity.default,
    strokeWidth: edgeDesignSystem.widths.default,
  },
  dimmed: {
    opacity: edgeDesignSystem.opacity.dimmed,
    strokeWidth: edgeDesignSystem.widths.dimmed,
  },
  highlighted: {
    opacity: edgeDesignSystem.opacity.highlighted,
    strokeWidth: edgeDesignSystem.widths.highlighted,
    filter: 'drop-shadow(0 0 2px currentColor)',
  },
  hover: {
    opacity: edgeDesignSystem.opacity.hover,
    strokeWidth: edgeDesignSystem.widths.hover,
    filter: 'drop-shadow(0 0 4px currentColor)',
    transition: edgeDesignSystem.animations.transition.duration + ' ' + edgeDesignSystem.animations.transition.easing,
  },
}

// Relationship label lookup - avoids switch statement
const RELATIONSHIP_LABELS: Record<string, string> = {
  hasOne: 'One-to-One',
  hasMany: 'One-to-Many',
  belongsTo: 'Many-to-One',
  manyToMany: 'Many-to-Many',
}

/**
 * CustomEdge - Classic mode edge with smooth step path
 *
 * Performance optimizations:
 * - Wrapped in React.memo with custom comparator
 * - Pre-computed style objects outside component
 * - Hover state only for tooltip (minimized state updates)
 * - Memoized edge style computation
 */
function CustomEdgeComponent({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  markerEnd,
  data,
}: EdgeProps<CustomEdgeData>) {
  // Track hover state and mouse position for tooltip
  // Note: Mouse position state only updates during hover, minimizing re-renders
  const [hoverState, setHoverState] = useState<{ isHovered: boolean; x: number; y: number }>({
    isHovered: false,
    x: 0,
    y: 0,
  })

  const edgeData = data?.data
  const isHighlighted = data?.isHighlighted || false
  const isDimmed = data?.isDimmed || false

  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  const handleMouseEnter = useCallback(
    (event: React.MouseEvent) => {
      setHoverState({ isHovered: true, x: event.clientX, y: event.clientY })
      data?.onEdgeHover?.(id)
    },
    [id, data]
  )

  const handleMouseLeave = useCallback(() => {
    setHoverState(prev => ({ ...prev, isHovered: false }))
    data?.onEdgeHover?.(null)
  }, [data])

  // Only update position when already hovered (avoids extra state updates)
  const handleMouseMove = useCallback((event: React.MouseEvent) => {
    setHoverState(prev => prev.isHovered ? { ...prev, x: event.clientX, y: event.clientY } : prev)
  }, [])

  // Memoize relationship type to avoid recalculation
  const relationshipType = useMemo(() => {
    if (!edgeData) return ''
    return getCardinalitySymbol(edgeData.relation)
  }, [edgeData])

  // Memoize relationship label using lookup table
  const relationshipLabel = useMemo(() => {
    if (!edgeData) return ''
    return RELATIONSHIP_LABELS[edgeData.relation] || ''
  }, [edgeData])

  // Memoize edge style - only recalculate when state changes
  const edgeStyle = useMemo(() => {
    if (isDimmed) {
      return { ...style, ...baseEdgeStyles.dimmed }
    }
    if (hoverState.isHovered) {
      return { ...style, ...baseEdgeStyles.hover }
    }
    if (isHighlighted) {
      return { ...style, ...baseEdgeStyles.highlighted }
    }
    return { ...style, ...baseEdgeStyles.default }
  }, [style, isDimmed, hoverState.isHovered, isHighlighted])

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={edgeStyle}
      />

      {/* Invisible wider path for easier hover detection */}
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={20}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onMouseMove={handleMouseMove}
        style={{ cursor: 'pointer' }}
      />

      {/* Enhanced tooltip on hover with design system styling */}
      {hoverState.isHovered && edgeData && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'fixed',
              left: hoverState.x + 10,
              top: hoverState.y + 10,
              pointerEvents: 'none',
              zIndex: 1000,
              ...edgeDesignSystem.labels.typography,
            }}
            className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border text-xs max-w-xs backdrop-blur-sm"
            role="tooltip"
            aria-live="polite"
          >
            <div className="font-semibold mb-1 flex items-center gap-2">
              <span>{relationshipLabel}</span>
              <span className="text-lg">{relationshipType}</span>
            </div>
            <div className="text-muted-foreground font-mono text-[11px]">
              {edgeData.sourceKey} {'->'} {edgeData.targetKey}
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              {edgeData.source.split('.').pop()} to {edgeData.target.split('.').pop()}
            </div>
          </div>
        </EdgeLabelRenderer>
      )}

      {/* Cardinality label with design system styling */}
      {edgeData && !isDimmed && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: 'none',
              background: edgeDesignSystem.labels.container.background,
              border: edgeDesignSystem.labels.container.border,
              borderRadius: edgeDesignSystem.labels.container.borderRadius,
              padding: edgeDesignSystem.labels.container.padding,
              boxShadow: edgeDesignSystem.labels.container.boxShadow,
              backdropFilter: edgeDesignSystem.labels.container.backdropFilter,
              fontSize: edgeDesignSystem.labels.typography.fontSize,
              fontWeight: edgeDesignSystem.labels.typography.fontWeight,
              letterSpacing: edgeDesignSystem.labels.typography.letterSpacing,
            }}
            className="text-foreground"
            aria-label={`${relationshipLabel} relationship`}
          >
            {relationshipType}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}

// Wrap in React.memo with custom comparator for optimal performance
// Only re-render when position or visual state changes
export const CustomEdge = memo(CustomEdgeComponent, (prevProps, nextProps) => {
  return (
    prevProps.sourceX === nextProps.sourceX &&
    prevProps.sourceY === nextProps.sourceY &&
    prevProps.targetX === nextProps.targetX &&
    prevProps.targetY === nextProps.targetY &&
    prevProps.data?.isHighlighted === nextProps.data?.isHighlighted &&
    prevProps.data?.isDimmed === nextProps.data?.isDimmed
  )
})
