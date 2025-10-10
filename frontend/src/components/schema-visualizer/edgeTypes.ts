import { Edge } from 'reactflow'

// Custom edge types for different relationship types
export const edgeTypes = {
  default: {
    style: {
      stroke: '#64748b',
      strokeWidth: 2,
    },
    markerEnd: {
      type: 'arrowclosed',
      color: '#64748b',
    },
  },
  primary: {
    style: {
      stroke: '#3b82f6',
      strokeWidth: 2,
    },
    markerEnd: {
      type: 'arrowclosed',
      color: '#3b82f6',
    },
  },
  foreign: {
    style: {
      stroke: '#8b5cf6',
      strokeWidth: 2,
      strokeDasharray: '5,5',
    },
    markerEnd: {
      type: 'arrowclosed',
      color: '#8b5cf6',
    },
  },
  oneToMany: {
    style: {
      stroke: '#f59e0b',
      strokeWidth: 2,
    },
    markerEnd: {
      type: 'arrowclosed',
      color: '#f59e0b',
    },
  },
  manyToMany: {
    style: {
      stroke: '#ef4444',
      strokeWidth: 2,
      strokeDasharray: '10,5',
    },
    markerEnd: {
      type: 'arrowclosed',
      color: '#ef4444',
    },
  },
}

// Custom edge component for better visualization
export function CustomEdge({ data, style, ...props }: Edge) {
  const edgeStyle = {
    ...style,
    strokeWidth: 2,
    strokeDasharray: data?.relation === 'hasMany' ? '5,5' : 'none',
  }

  return (
    <g>
      <path
        style={edgeStyle}
        d={props.path}
        markerEnd="url(#arrowhead)"
        className="react-flow__edge-path"
      />
      {data?.label && (
        <text>
          <textPath href={`#${props.id}`} style={{ fontSize: 12, fill: '#666' }}>
            {data.label}
          </textPath>
        </text>
      )}
    </g>
  )
}
