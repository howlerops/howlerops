import React, { memo, useCallback } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  EdgeProps,
  getBezierPath,
} from 'reactflow'

import { cn } from '@/lib/utils'
import { EdgeConfig } from '@/types/schema-visualizer'

interface ERDEdgeData {
  data?: EdgeConfig
  onEdgeHover?: (edgeId: string | null) => void
  isHighlighted?: boolean
  isDimmed?: boolean
}

// Pre-computed styles to avoid object recreation
const BASE_STROKE = '#64748b'
const HIGHLIGHT_STROKE = '#334155'

const baseEdgeStyle = {
  stroke: BASE_STROKE,
  strokeWidth: 3,
}

const dimmedEdgeStyle = {
  stroke: BASE_STROKE,
  strokeWidth: 2,
  opacity: 0.2,
}

const highlightedEdgeStyle = {
  stroke: HIGHLIGHT_STROKE,
  strokeWidth: 4,
}

// Relationship info lookup table - avoids switch statement
const RELATION_INFO: Record<string, { sourceCardinality: string; targetCardinality: string; label: string }> = {
  hasOne: { sourceCardinality: 'one', targetCardinality: 'one', label: 'One-to-One' },
  hasMany: { sourceCardinality: 'many', targetCardinality: 'one', label: 'One-to-Many' },
  belongsTo: { sourceCardinality: 'one', targetCardinality: 'one', label: 'Belongs To' },
  manyToMany: { sourceCardinality: 'many', targetCardinality: 'many', label: 'Many-to-Many' },
}

const DEFAULT_RELATION_INFO = { sourceCardinality: 'one', targetCardinality: 'one', label: '' }

// Cardinality symbols lookup
const CARDINALITY_SYMBOLS: Record<string, string> = {
  hasOne: '1:1',
  hasMany: '1:N',
  belongsTo: 'N:1',
  manyToMany: 'N:M',
}

/**
 * ERD Edge with crow's foot notation - Performance optimized
 *
 * Optimizations:
 * - Removed useState for hover state (causes re-renders during mouse move)
 * - Hover tooltip only shown via CSS :hover pseudo-class
 * - Marker SVGs use CSS transforms instead of React state
 * - Memoized with React.memo
 * - Pre-computed styles outside component
 */
function ERDEdgeComponent({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps<ERDEdgeData>) {
  const edgeData = data?.data
  const isHighlighted = data?.isHighlighted || false
  const isDimmed = data?.isDimmed || false

  // Calculate bezier path for smooth curves
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    curvature: 0.25,
  })

  // Get relationship info without useMemo (lookup is fast)
  const relationInfo = edgeData?.relation
    ? RELATION_INFO[edgeData.relation] || DEFAULT_RELATION_INFO
    : DEFAULT_RELATION_INFO

  // Select pre-computed style based on state
  const edgeStyle = isDimmed
    ? dimmedEdgeStyle
    : isHighlighted
      ? highlightedEdgeStyle
      : baseEdgeStyle

  // Simple rotation calculation (no useMemo needed for simple math)
  const sourceMarkerRotation = targetX > sourceX ? 0 : 180
  const targetMarkerRotation = sourceX > targetX ? 0 : 180

  // Hover handlers that update parent state (debounced in parent)
  const handleMouseEnter = useCallback(() => {
    data?.onEdgeHover?.(id)
  }, [id, data])

  const handleMouseLeave = useCallback(() => {
    data?.onEdgeHover?.(null)
  }, [data])

  // Use CSS-based stroke color for markers
  const markerColor = isDimmed ? BASE_STROKE : (isHighlighted ? HIGHLIGHT_STROKE : BASE_STROKE)

  return (
    <g className="erd-edge-group">
      {/* Main edge path */}
      <BaseEdge
        id={id}
        path={edgePath}
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
        style={{ cursor: 'pointer' }}
        className="erd-edge-hitarea"
      />

      {/* Source crow's foot marker - simplified SVG */}
      {!isDimmed && (
        <g
          transform={`translate(${sourceX - 5}, ${sourceY}) rotate(${sourceMarkerRotation})`}
          style={{ pointerEvents: 'none' }}
        >
          {relationInfo.sourceCardinality === 'many' ? (
            <>
              <line x1="0" y1="-6" x2="10" y2="0" stroke={markerColor} strokeWidth="1.5" />
              <line x1="0" y1="0" x2="10" y2="0" stroke={markerColor} strokeWidth="1.5" />
              <line x1="0" y1="6" x2="10" y2="0" stroke={markerColor} strokeWidth="1.5" />
            </>
          ) : (
            <line x1="0" y1="0" x2="10" y2="0" stroke={markerColor} strokeWidth="2" />
          )}
        </g>
      )}

      {/* Target crow's foot marker - simplified SVG */}
      {!isDimmed && (
        <g
          transform={`translate(${targetX - 10}, ${targetY}) rotate(${targetMarkerRotation})`}
          style={{ pointerEvents: 'none' }}
        >
          {relationInfo.targetCardinality === 'many' ? (
            <>
              <line x1="0" y1="-6" x2="10" y2="0" stroke={markerColor} strokeWidth="1.5" />
              <line x1="0" y1="0" x2="10" y2="0" stroke={markerColor} strokeWidth="1.5" />
              <line x1="0" y1="6" x2="10" y2="0" stroke={markerColor} strokeWidth="1.5" />
            </>
          ) : (
            <line x1="0" y1="0" x2="10" y2="0" stroke={markerColor} strokeWidth="2" />
          )}
        </g>
      )}

      {/* Relationship indicator at midpoint - only when highlighted or not dimmed */}
      {edgeData && !isDimmed && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
              pointerEvents: 'none',
            }}
            className={cn(
              'bg-muted/90 border border-border rounded px-1.5 py-0.5',
              'text-[10px] text-muted-foreground font-medium',
              isHighlighted && 'bg-accent/90 text-accent-foreground'
            )}
          >
            {CARDINALITY_SYMBOLS[edgeData.relation] || '1:1'}
          </div>
        </EdgeLabelRenderer>
      )}
    </g>
  )
}

// Memoize the entire component to prevent unnecessary re-renders
export const ERDEdge = memo(ERDEdgeComponent, (prevProps, nextProps) => {
  // Custom comparison - only re-render when these values change
  return (
    prevProps.sourceX === nextProps.sourceX &&
    prevProps.sourceY === nextProps.sourceY &&
    prevProps.targetX === nextProps.targetX &&
    prevProps.targetY === nextProps.targetY &&
    prevProps.data?.isHighlighted === nextProps.data?.isHighlighted &&
    prevProps.data?.isDimmed === nextProps.data?.isDimmed
  )
})
