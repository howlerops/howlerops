import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Database, Network, ChevronDown, Info, CheckCircle2, Circle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { useConnectionStore } from '@/store/connection-store'
import { UseQueryModeReturn } from '@/hooks/useQueryMode'

interface ModeSwitcherProps {
  mode: UseQueryModeReturn['mode']
  canToggle?: UseQueryModeReturn['canToggle']
  toggleMode: UseQueryModeReturn['toggleMode']
  connectionCount?: UseQueryModeReturn['connectionCount']
  className?: string
}

interface ModeConfig {
  key: 'single' | 'multi'
  label: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  color: {
    bg: string
    border: string
    text: string
    icon: string
    badge: string
  }
  requirements: string[]
}

const modeConfigs: Record<'single' | 'multi', ModeConfig> = {
  single: {
    key: 'single',
    label: 'Single Database',
    description: 'Query one database at a time with full autocomplete support',
    icon: Database,
    color: {
      bg: 'bg-blue-50 dark:bg-blue-950/30',
      border: 'border-blue-200 dark:border-blue-800',
      text: 'text-blue-900 dark:text-blue-100',
      icon: 'text-blue-600 dark:text-blue-400',
      badge: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    },
    requirements: ['At least 1 connection configured'],
  },
  multi: {
    key: 'multi',
    label: 'Multi-Database',
    description: 'Query across multiple databases using @connectionName.table syntax',
    icon: Network,
    color: {
      bg: 'bg-purple-50 dark:bg-purple-950/30',
      border: 'border-purple-200 dark:border-purple-800',
      text: 'text-purple-900 dark:text-purple-100',
      icon: 'text-purple-600 dark:text-purple-400',
      badge: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
    },
    requirements: ['At least 2 connections configured', 'Connections must be in same environment'],
  },
}

export function ModeSwitcher({ mode, canToggle, toggleMode, connectionCount, className }: ModeSwitcherProps) {
  const { connections, getFilteredConnections, activeEnvironmentFilter } = useConnectionStore()
  const [isOpen, setIsOpen] = useState(false)

  const currentConfig = modeConfigs[mode]
  const filteredConnections = getFilteredConnections()
  const connectedCount = filteredConnections.filter(c => c.isConnected).length

  // Check if multi-mode requirements are met
  const canUseMulti = filteredConnections.length >= 2

  useEffect(() => {
    // Auto-close dropdown when mode changes
    setIsOpen(false)
  }, [mode])

  // Auto-connect filtered connections when switching to multi-DB mode
  useEffect(() => {
    const autoConnectForMultiDB = async () => {
      if (mode === 'multi') {
        const filtered = getFilteredConnections()
        const disconnected = filtered.filter(c => !c.isConnected)
        
        if (disconnected.length > 0) {
          console.log(`⚡ Multi-DB mode: Auto-connecting ${disconnected.length} connections...`)
          
          const { connectToDatabase } = useConnectionStore.getState()
          const connectPromises = disconnected.map(async (conn) => {
            try {
              await connectToDatabase(conn.id)
              console.log(`  ✓ Connected: ${conn.name}`)
            } catch (error) {
              console.warn(`  ✗ Failed: ${conn.name}`, error)
            }
          })
          
          await Promise.allSettled(connectPromises)
        }
      }
    }
    
    autoConnectForMultiDB()
  }, [mode, getFilteredConnections])

  const handleModeChange = (newMode: 'single' | 'multi') => {
    if (newMode === mode) return

    if (newMode === 'multi' && !canUseMulti) {
      return // Can't switch to multi if requirements not met
    }

    toggleMode()
  }

  return (
    <div className={cn('flex items-center', className)}>
      <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            className={cn(
              'h-9 px-3 gap-2 transition-all duration-200 hover:shadow-sm',
              currentConfig.color.border,
              currentConfig.color.bg,
              currentConfig.color.text,
              'hover:scale-[1.02]'
            )}
          >
            <motion.div
              key={mode}
              initial={{ rotate: -10, scale: 0.8 }}
              animate={{ rotate: 0, scale: 1 }}
              transition={{ duration: 0.2, ease: 'easeOut' }}
            >
              <currentConfig.icon className={cn('h-4 w-4', currentConfig.color.icon)} />
            </motion.div>

            <div className="flex items-center gap-2">
              <span className="font-medium">{currentConfig.label}</span>
              <Badge
                variant="secondary"
                className={cn('h-5 px-1.5 text-xs', currentConfig.color.badge)}
              >
                {mode === 'multi' ? `${filteredConnections.length} DBs` : '1 DB'}
              </Badge>
            </div>

            <motion.div
              animate={{ rotate: isOpen ? 180 : 0 }}
              transition={{ duration: 0.2 }}
            >
              <ChevronDown className="h-3 w-3 opacity-60" />
            </motion.div>
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="start" className="w-80 p-0">
          <div className="p-3 border-b">
            <div className="flex items-center gap-2 mb-2">
              <Info className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium text-sm">Query Mode</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Choose how you want to query your databases
            </p>
          </div>

          {/* Mode Options */}
          <div className="p-1">
            {Object.values(modeConfigs).map((config) => {
              const isActive = mode === config.key
              const isDisabled = config.key === 'multi' && !canUseMulti

              return (
                <DropdownMenuItem
                  key={config.key}
                  className={cn(
                    'p-3 cursor-pointer transition-all duration-150',
                    isActive && config.color.bg,
                    isActive && config.color.border + ' border',
                    isDisabled && 'opacity-50 cursor-not-allowed'
                  )}
                  onSelect={() => !isDisabled && handleModeChange(config.key)}
                  disabled={isDisabled}
                >
                  <div className="flex items-start gap-3 w-full">
                    <div className={cn(
                      'rounded-lg p-2 transition-colors',
                      isActive ? config.color.bg : 'bg-muted/50'
                    )}>
                      <config.icon className={cn(
                        'h-4 w-4',
                        isActive ? config.color.icon : 'text-muted-foreground'
                      )} />
                    </div>

                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={cn(
                          'font-medium text-sm',
                          isActive && config.color.text
                        )}>
                          {config.label}
                        </span>
                        {isActive && (
                          <motion.div
                            initial={{ scale: 0 }}
                            animate={{ scale: 1 }}
                            transition={{ duration: 0.2 }}
                          >
                            <CheckCircle2 className={cn('h-4 w-4', config.color.icon)} />
                          </motion.div>
                        )}
                      </div>

                      <p className="text-xs text-muted-foreground leading-relaxed">
                        {config.description}
                      </p>

                      {/* Requirements */}
                      <div className="mt-2 space-y-1">
                        {config.requirements.map((req, index) => {
                          let isMet = false

                          if (req.includes('At least 1 connection')) {
                            isMet = connections.length >= 1
                          } else if (req.includes('At least 2 connections')) {
                            isMet = filteredConnections.length >= 2
                          } else if (req.includes('same environment')) {
                            isMet = true // Environment filtering handles this
                          }

                          return (
                            <div key={index} className="flex items-center gap-1.5">
                              {isMet ? (
                                <CheckCircle2 className="h-3 w-3 text-green-500" />
                              ) : (
                                <Circle className="h-3 w-3 text-muted-foreground" />
                              )}
                              <span className={cn(
                                'text-xs',
                                isMet ? 'text-green-700 dark:text-green-400' : 'text-muted-foreground'
                              )}>
                                {req}
                              </span>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  </div>
                </DropdownMenuItem>
              )
            })}
          </div>

          <DropdownMenuSeparator />

          {/* Status Information */}
          <div className="p-3 bg-muted/30">
            <div className="grid grid-cols-2 gap-3 text-xs">
              <div>
                <span className="text-muted-foreground">Total Connections:</span>
                <span className="font-medium ml-1">{connections.length}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Connected:</span>
                <span className="font-medium ml-1 text-green-600 dark:text-green-400">
                  {connectedCount}
                </span>
              </div>
              {activeEnvironmentFilter && (
                <div className="col-span-2">
                  <span className="text-muted-foreground">Environment:</span>
                  <Badge variant="outline" className="ml-1 h-4 text-xs">
                    {activeEnvironmentFilter}
                  </Badge>
                </div>
              )}
            </div>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}