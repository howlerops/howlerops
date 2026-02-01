import { Loader2 } from 'lucide-react'

import { Button } from '../../ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../ui/dialog'
import type { DeleteConfirmDialogProps } from '../types'

export function DeleteConfirmDialog({
  open,
  onOpenChange,
  pendingDeleteCount,
  isDeleting,
  onConfirm,
  onCancel,
}: DeleteConfirmDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (isDeleting) return
        onOpenChange(nextOpen)
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete selected rows?</DialogTitle>
          <DialogDescription>
            {pendingDeleteCount === 1
              ? 'This will permanently delete the selected row.'
              : `This will permanently delete ${pendingDeleteCount} rows.`}
            {' '}This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={onCancel}
            disabled={isDeleting}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={isDeleting}
            className="gap-2"
          >
            {isDeleting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Deleting...
              </>
            ) : (
              `Delete ${pendingDeleteCount} row${pendingDeleteCount === 1 ? '' : 's'}`
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
