import { memo } from 'react'

import type { AgentAttachment } from '@/store/ai-query-agent-store'

import { ChartAttachment } from './ChartAttachment'
import { InsightAttachment } from './InsightAttachment'
import { ReportAttachment } from './ReportAttachment'
import { ResultAttachment } from './ResultAttachment'
import { SQLAttachment } from './SQLAttachment'

interface AttachmentRendererProps {
  attachment: AgentAttachment
  index: number
  onCopySQL?: (sql: string) => void
  onUseSQL?: (sql: string, connectionId?: string) => void
  onExportResult?: (columns: string[], rows: Record<string, unknown>[]) => void
}

export const AttachmentRenderer = memo(({
  attachment,
  index,
  onCopySQL,
  onUseSQL,
  onExportResult,
}: AttachmentRendererProps) => {
  switch (attachment.type) {
    case 'sql':
      return attachment.sql ? (
        <SQLAttachment
          key={`sql-${index}`}
          sql={attachment.sql}
          onCopy={onCopySQL}
          onUse={onUseSQL}
        />
      ) : null

    case 'result':
      return attachment.result ? (
        <ResultAttachment
          key={`result-${index}`}
          result={attachment.result}
          onExport={onExportResult}
        />
      ) : null

    case 'chart':
      return attachment.chart ? (
        <ChartAttachment
          key={`chart-${index}`}
          chart={attachment.chart}
        />
      ) : null

    case 'report':
      return attachment.report ? (
        <ReportAttachment
          key={`report-${index}`}
          report={attachment.report}
        />
      ) : null

    case 'insight':
      return attachment.insight ? (
        <InsightAttachment
          key={`insight-${index}`}
          insight={attachment.insight}
        />
      ) : null

    default:
      return null
  }
})

AttachmentRenderer.displayName = 'AttachmentRenderer'
