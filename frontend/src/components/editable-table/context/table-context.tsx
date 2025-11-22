import { createContext, useContext } from 'react';

import type {
  CellValue,
  EditableTableActions,
  TableRow,
  TableState,
} from '../../../types/table';

export interface TableContextValue {
  // State & Actions (frequently changing)
  state: TableState;
  actions: EditableTableActions;

  // Callbacks (stable references)
  onRowInspect?: (rowId: string, rowData: TableRow) => void;

  // Custom renderers (stable)
  customCellRenderers: Record<string, (value: CellValue, row: TableRow) => React.ReactNode>;
}

export const TableContext = createContext<TableContextValue | null>(null);

export const useTableContext = (): TableContextValue => {
  const context = useContext(TableContext);
  if (!context) {
    throw new Error('useTableContext must be used within TableContext.Provider');
  }
  return context;
};
