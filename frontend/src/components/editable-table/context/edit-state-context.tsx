import { createContext, useContext } from 'react';

import type { CellEditState, TableAction } from '../../../types/table';

export interface EditStateValue {
  editingCell: CellEditState | null;
  invalidCells: ReadonlyMap<string, { columnId: string; error: string }>;
  dirtyRows: ReadonlySet<string>;
  undoStack: readonly TableAction[];
  redoStack: readonly TableAction[];
  // Computed values
  hasUndoActions: boolean;
  hasRedoActions: boolean;
}

export const EditStateContext = createContext<EditStateValue | null>(null);

export const useEditState = (): EditStateValue => {
  const context = useContext(EditStateContext);
  if (!context) {
    throw new Error('useEditState must be used within EditStateContext.Provider');
  }
  return context;
};
