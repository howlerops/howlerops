import { createContext, useContext } from 'react';

import type { CellValue,TableRow } from '../../../types/table';

export interface TableConfigValue {
  // Stable references (rarely change)
  onRowInspect?: (rowId: string, rowData: TableRow) => void;
  customCellRenderers: Record<string, (value: CellValue, row: TableRow) => React.ReactNode>;
}

export const TableConfigContext = createContext<TableConfigValue | null>(null);

export const useTableConfig = (): TableConfigValue => {
  const context = useContext(TableConfigContext);
  if (!context) {
    throw new Error('useTableConfig must be used within TableConfigContext.Provider');
  }
  return context;
};
