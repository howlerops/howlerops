import { useCallback, useState } from 'react'
import { CellValue } from '@/types/table'

export function getValueClass(value: CellValue): string {
  if (value === null) return 'text-gray-500'
  if (typeof value === 'boolean') return 'text-orange-600'
  if (typeof value === 'number') return 'text-purple-600'
  if (typeof value === 'string') return 'text-green-600'
  if (typeof value === 'object') return 'text-gray-700'
  return 'text-gray-900'
}

export function formatValue(value: CellValue): string {
  if (value === null) return 'null'
  if (value === undefined) return 'undefined'
  if (typeof value === 'boolean') return value.toString()
  if (typeof value === 'number') return value.toString()
  if (typeof value === 'string') return `"${value}"`
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return '[Object]'
    }
  }
  return String(value)
}

export function useJsonEditor() {
  const [isEditing, setIsEditing] = useState(false)
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set())
  const [collapsedKeys, setCollapsedKeys] = useState<Set<string>>(new Set())

  const toggleEdit = useCallback(() => {
    setIsEditing(prev => !prev)
  }, [])

  const toggleKeyExpansion = useCallback((key: string) => {
    setExpandedKeys(prev => {
      const newSet = new Set(prev)
      if (newSet.has(key)) {
        newSet.delete(key)
        setCollapsedKeys(prevCollapsed => new Set(prevCollapsed).add(key))
      } else {
        newSet.add(key)
        setCollapsedKeys(prevCollapsed => {
          const newCollapsed = new Set(prevCollapsed)
          newCollapsed.delete(key)
          return newCollapsed
        })
      }
      return newSet
    })
  }, [])

  const expandAllKeys = useCallback(() => {
    setExpandedKeys(new Set(['*']))
    setCollapsedKeys(new Set())
  }, [])

  const collapseAllKeys = useCallback(() => {
    setExpandedKeys(new Set())
    setCollapsedKeys(new Set(['*']))
  }, [])

  return {
    isEditing,
    expandedKeys,
    collapsedKeys,
    toggleEdit,
    toggleKeyExpansion,
    expandAllKeys,
    collapseAllKeys,
  }
}
