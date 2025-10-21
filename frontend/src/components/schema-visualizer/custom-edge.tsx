import React, { useState, useCallback } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  EdgeProps,
  getSmoothStepPath,
} from 'reactflow'
import { EdgeConfig } from '@/types/schema-visualizer'
import { edgeDesignSystem, getCardinalitySymbol } from '@/lib/edge-design-tokens'

interface CustomEdgeData {
  data?: EdgeConfig
  onEdgeHover?: (edgeId: string | null) => void
  isHighlighted?: boolean
  isDimmed?: boolean
}

export function CustomEdge({
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
  const [isHovered, setIsHovered] = useState(false)
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 })

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
      setIsHovered(true)
      setMousePosition({ x: event.clientX, y: event.clientY })
      if (data?.onEdgeHover) {
        data.onEdgeHover(id)
      }
    },
    [id, data]
  )

  const handleMouseLeave = useCallback(() => {
    setIsHovered(false)
    if (data?.onEdgeHover) {
      data.onEdgeHover(null)
    }
  }, [data])

  const handleMouseMove = useCallback((event: React.MouseEvent) => {
    setMousePosition({ x: event.clientX, y: event.clientY })
  }, [])

  // Calculate relationship type display using design tokens
  const getRelationshipType = () => {
    if (!edgeData) return ''
    return getCardinalitySymbol(edgeData.relation)
  }

  // Get relationship type label
  const getRelationshipLabel = () => {
    if (!edgeData) return ''

    switch (edgeData.relation) {
      case 'hasOne':
        return 'One-to-One'
      case 'hasMany':
        return 'One-to-Many'
      case 'belongsTo':
        return 'Many-to-One'
      case 'manyToMany':
        return 'Many-to-Many'
      default:
        return ''
    }
  }

  // Determine edge styling based on state using design tokens
  const getEdgeStyle = () => {
    const baseStyle = { ...style }
    const tokens = edgeDesignSystem

    if (isDimmed) {
      return {
        ...baseStyle,
        opacity: tokens.opacity.dimmed,
        strokeWidth: tokens.widths.dimmed,
      }
    }

    if (isHovered) {
      return {
        ...baseStyle,
        opacity: tokens.opacity.hover,
        strokeWidth: tokens.widths.hover,
        filter: 'drop-shadow(0 0 4px currentColor)',
        transition: tokens.animations.transition.duration + ' ' + tokens.animations.transition.easing,
      }
    }

    if (isHighlighted) {
      return {
        ...baseStyle,
        opacity: tokens.opacity.highlighted,
        strokeWidth: tokens.widths.highlighted,
        filter: 'drop-shadow(0 0 2px currentColor)',
      }
    }

    return {
      ...baseStyle,
      opacity: tokens.opacity.default,
      strokeWidth: tokens.widths.default,
    }
  }

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={getEdgeStyle()}
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
      {isHovered && edgeData && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'fixed',
              left: mousePosition.x + 10,
              top: mousePosition.y + 10,
              pointerEvents: 'none',
              zIndex: 1000,
              ...edgeDesignSystem.labels.typography,
            }}
            className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border text-xs max-w-xs backdrop-blur-sm"
            role="tooltip"
            aria-live="polite"
          >
            <div className="font-semibold mb-1 flex items-center gap-2">
              <span>{getRelationshipLabel()}</span>
              <span className="text-lg">{getRelationshipType()}</span>
            </div>
            <div className="text-muted-foreground font-mono text-[11px]">
              {edgeData.sourceKey} → {edgeData.targetKey}
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
            aria-label={`${getRelationshipLabel()} relationship`}
          >
            {getRelationshipType()}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}
