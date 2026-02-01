import { Download, Loader2 } from 'lucide-react'
import { useState } from 'react'

import { Button } from '../../ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../ui/dialog'
import type { ExportButtonProps, ExportOptions } from '../types'

export function ExportButton({ context, onExport }: ExportButtonProps) {
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [isExporting, setIsExporting] = useState(false)
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'csv',
    includeHeaders: true,
    selectedOnly: false,
  })

  const handleExport = async () => {
    setIsExporting(true)
    try {
      await onExport(exportOptions)
      // Keep dialog open briefly to show success message (handled via toast)
      setTimeout(() => setShowExportDialog(false), 500)
    } catch {
      // Error is handled by onExport, just reset state
      setIsExporting(false)
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowExportDialog(true)}
        className="gap-2"
      >
        <Download className="h-4 w-4" />
        Export
      </Button>

      <Dialog open={showExportDialog} onOpenChange={setShowExportDialog}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Export Data</DialogTitle>
            <DialogDescription>
              Export will fetch ALL results from the database (up to 1M rows). Configure options and download to your Downloads folder.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Format selection */}
            <div className="space-y-2">
              <label className="text-sm font-medium">Format</label>
              <select
                value={exportOptions.format}
                onChange={(e) => setExportOptions(prev => ({ ...prev, format: e.target.value as 'csv' | 'json' }))}
                className="w-full px-3 py-2 border border-input rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-ring"
              >
                <option value="csv">CSV</option>
                <option value="json">JSON</option>
              </select>
            </div>

            {/* Options */}
            <div className="space-y-3">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={exportOptions.includeHeaders}
                  onChange={(e) => setExportOptions(prev => ({ ...prev, includeHeaders: e.target.checked }))}
                  className="rounded border-input focus:ring-2 focus:ring-ring"
                />
                <span className="text-sm">Include headers</span>
              </label>
              {context.state.selectedRows.length > 0 && (
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={exportOptions.selectedOnly}
                    onChange={(e) => setExportOptions(prev => ({ ...prev, selectedOnly: e.target.checked }))}
                    className="rounded border-input focus:ring-2 focus:ring-ring"
                  />
                  <span className="text-sm">
                    Selected only ({context.state.selectedRows.length} rows)
                  </span>
                </label>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowExportDialog(false)}
              disabled={isExporting}
            >
              Cancel
            </Button>
            <Button onClick={handleExport} disabled={isExporting}>
              {isExporting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Exporting...
                </>
              ) : (
                <>
                  <Download className="h-4 w-4 mr-2" />
                  Export to Downloads
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
