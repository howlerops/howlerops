/**
 * Cron Expression Utilities
 * Parse and generate human-readable cron expressions
 */

export interface CronSchedule {
  minute: string
  hour: string
  dayOfMonth: string
  month: string
  dayOfWeek: string
}

export interface CronPreset {
  label: string
  description: string
  expression: string
}

export const CRON_PRESETS: CronPreset[] = [
  {
    label: 'Every Hour',
    description: 'Runs at the start of every hour',
    expression: '0 * * * *',
  },
  {
    label: 'Every 6 Hours',
    description: 'Runs every 6 hours starting at midnight',
    expression: '0 */6 * * *',
  },
  {
    label: 'Daily at 9 AM',
    description: 'Runs once per day at 9:00 AM',
    expression: '0 9 * * *',
  },
  {
    label: 'Daily at Midnight',
    description: 'Runs once per day at 12:00 AM',
    expression: '0 0 * * *',
  },
  {
    label: 'Weekly on Monday',
    description: 'Runs every Monday at 9:00 AM',
    expression: '0 9 * * 1',
  },
  {
    label: 'Monthly on 1st',
    description: 'Runs on the 1st of every month at 9:00 AM',
    expression: '0 9 1 * *',
  },
  {
    label: 'Weekdays at 9 AM',
    description: 'Runs Monday through Friday at 9:00 AM',
    expression: '0 9 * * 1-5',
  },
]

/**
 * Parse a cron expression into its components
 */
export function parseCronExpression(expression: string): CronSchedule {
  const parts = expression.split(' ')

  if (parts.length !== 5) {
    throw new Error('Invalid cron expression. Expected 5 parts.')
  }

  return {
    minute: parts[0],
    hour: parts[1],
    dayOfMonth: parts[2],
    month: parts[3],
    dayOfWeek: parts[4],
  }
}

/**
 * Build a cron expression from its components
 */
export function buildCronExpression(schedule: CronSchedule): string {
  return [
    schedule.minute,
    schedule.hour,
    schedule.dayOfMonth,
    schedule.month,
    schedule.dayOfWeek,
  ].join(' ')
}

/**
 * Convert cron expression to human-readable description
 */
export function cronToHumanReadable(expression: string): string {
  try {
    const cron = parseCronExpression(expression)

    // Check for presets
    const preset = CRON_PRESETS.find((p) => p.expression === expression)
    if (preset) {
      return preset.description
    }

    // Build description
    const parts: string[] = []

    // Frequency
    if (cron.minute === '*' && cron.hour === '*') {
      parts.push('Runs every minute')
    } else if (cron.minute.startsWith('*/')) {
      const interval = cron.minute.slice(2)
      parts.push(`Runs every ${interval} minutes`)
    } else if (cron.hour.startsWith('*/')) {
      const interval = cron.hour.slice(2)
      parts.push(`Runs every ${interval} hours`)
    } else if (cron.dayOfMonth !== '*' && cron.month !== '*') {
      parts.push(`Runs on day ${cron.dayOfMonth} of month ${cron.month}`)
    } else if (cron.dayOfWeek !== '*') {
      const day = getDayName(cron.dayOfWeek)
      parts.push(`Runs every ${day}`)
    } else if (cron.dayOfMonth !== '*') {
      parts.push(`Runs on day ${cron.dayOfMonth} of every month`)
    } else {
      parts.push('Runs daily')
    }

    // Time
    if (cron.hour !== '*' && cron.minute !== '*') {
      const time = formatTime(cron.hour, cron.minute)
      parts.push(`at ${time}`)
    }

    return parts.join(' ')
  } catch (error) {
    return 'Invalid cron expression'
  }
}

/**
 * Get next scheduled run time
 */
export function getNextRunTime(expression: string, from: Date = new Date()): Date {
  // This is a simplified version - in production use a library like cron-parser
  const cron = parseCronExpression(expression)
  const next = new Date(from)

  // Simple logic for common cases
  if (cron.minute !== '*') {
    next.setMinutes(parseInt(cron.minute, 10))
  }
  if (cron.hour !== '*') {
    next.setHours(parseInt(cron.hour, 10))
  }

  // If time has passed today, move to tomorrow
  if (next <= from) {
    next.setDate(next.getDate() + 1)
  }

  return next
}

/**
 * Validate cron expression
 */
export function isValidCronExpression(expression: string): boolean {
  try {
    const parts = expression.split(' ')
    if (parts.length !== 5) return false

    const validators = [
      (v: string) => isValidCronValue(v, 0, 59), // minute
      (v: string) => isValidCronValue(v, 0, 23), // hour
      (v: string) => isValidCronValue(v, 1, 31), // day of month
      (v: string) => isValidCronValue(v, 1, 12), // month
      (v: string) => isValidCronValue(v, 0, 6), // day of week
    ]

    return parts.every((part, i) => validators[i](part))
  } catch {
    return false
  }
}

function isValidCronValue(value: string, min: number, max: number): boolean {
  // Allow wildcards
  if (value === '*') return true

  // Allow ranges (e.g., 1-5)
  if (value.includes('-')) {
    const [start, end] = value.split('-').map((v) => parseInt(v, 10))
    return start >= min && end <= max && start <= end
  }

  // Allow intervals (e.g., */5)
  if (value.startsWith('*/')) {
    const interval = parseInt(value.slice(2), 10)
    return interval > 0 && interval <= max
  }

  // Allow lists (e.g., 1,3,5)
  if (value.includes(',')) {
    return value.split(',').every((v) => {
      const num = parseInt(v, 10)
      return num >= min && num <= max
    })
  }

  // Single value
  const num = parseInt(value, 10)
  return num >= min && num <= max
}

function getDayName(dayOfWeek: string): string {
  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

  if (dayOfWeek === '*') return 'day'

  const num = parseInt(dayOfWeek, 10)
  if (num >= 0 && num <= 6) {
    return days[num]
  }

  return 'day'
}

function formatTime(hour: string, minute: string): string {
  const h = parseInt(hour, 10)
  const m = parseInt(minute, 10)

  const period = h >= 12 ? 'PM' : 'AM'
  const displayHour = h === 0 ? 12 : h > 12 ? h - 12 : h
  const displayMinute = m.toString().padStart(2, '0')

  return `${displayHour}:${displayMinute} ${period}`
}

/**
 * Get a friendly relative time description for next run
 */
export function getRelativeNextRun(nextRunAt: string | Date): string {
  const next = typeof nextRunAt === 'string' ? new Date(nextRunAt) : nextRunAt
  const now = new Date()
  const diffMs = next.getTime() - now.getTime()

  if (diffMs < 0) return 'Overdue'

  const minutes = Math.floor(diffMs / 60000)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) return `in ${days} day${days > 1 ? 's' : ''}`
  if (hours > 0) return `in ${hours} hour${hours > 1 ? 's' : ''}`
  if (minutes > 0) return `in ${minutes} minute${minutes > 1 ? 's' : ''}`
  return 'in less than a minute'
}
