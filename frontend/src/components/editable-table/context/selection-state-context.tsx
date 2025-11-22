import { createContext, useContext } from 'react';

export interface SelectionStateValue {
  selectedRows: readonly string[];
  selectAllPagesMode: boolean;
  // Computed values
  selectedCount: number;
}

export const SelectionStateContext = createContext<SelectionStateValue | null>(null);

export const useSelectionState = (): SelectionStateValue => {
  const context = useContext(SelectionStateContext);
  if (!context) {
    throw new Error('useSelectionState must be used within SelectionStateContext.Provider');
  }
  return context;
};
