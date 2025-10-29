import { useState, useCallback, useEffect, useRef } from 'react';
import { KeyboardNavigationState, ClipboardData } from '../types/table';
import { copyToClipboard, pasteFromClipboard } from '../utils/table';

interface UseKeyboardNavigationProps {
  rowCount: number;
  columnCount: number;
  onCellFocus?: (rowIndex: number, columnIndex: number) => void;
  onCellEdit?: (rowIndex: number, columnIndex: number) => void;
  onCopy?: (data: ClipboardData) => void;
  onPaste?: (data: ClipboardData, startRow: number, startCol: number) => void;
  onDelete?: (rowIndex: number, columnIndex: number) => void;
  onUndo?: () => void;
  onRedo?: () => void;
  disabled?: boolean;
}

export const useKeyboardNavigation = ({
  rowCount,
  columnCount,
  onCellFocus,
  onCellEdit,
  onCopy,
  onPaste,
  onDelete,
  onUndo,
  onRedo,
  disabled = false,
}: UseKeyboardNavigationProps) => {
  const [navigationState, setNavigationState] = useState<KeyboardNavigationState>({
    focusedCell: null,
    isEditing: false,
    selection: null,
  });

  const containerRef = useRef<HTMLDivElement>(null);
  const isComposingRef = useRef(false);

  const focusCell = useCallback((rowIndex: number, columnIndex: number) => {
    if (disabled || rowIndex < 0 || rowIndex >= rowCount || columnIndex < 0 || columnIndex >= columnCount) {
      return;
    }

    setNavigationState(prev => ({
      ...prev,
      focusedCell: { rowIndex, columnIndex },
      selection: null,
    }));

    onCellFocus?.(rowIndex, columnIndex);
  }, [disabled, rowCount, columnCount, onCellFocus]);

  const startEditing = useCallback(() => {
    if (disabled || !navigationState.focusedCell) {
      return;
    }

    setNavigationState(prev => ({
      ...prev,
      isEditing: true,
    }));

    const { rowIndex, columnIndex } = navigationState.focusedCell;
    onCellEdit?.(rowIndex, columnIndex);
  }, [disabled, navigationState.focusedCell, onCellEdit]);

  const stopEditing = useCallback(() => {
    setNavigationState(prev => ({
      ...prev,
      isEditing: false,
    }));
  }, []);

  const handleCopy = useCallback(async () => {
    if (disabled || !navigationState.focusedCell) {
      return;
    }

    const { selection, focusedCell } = navigationState;

    let startRow: number, endRow: number, startCol: number, endCol: number;

    if (selection) {
      startRow = Math.min(selection.start.rowIndex, selection.end.rowIndex);
      endRow = Math.max(selection.start.rowIndex, selection.end.rowIndex);
      startCol = Math.min(selection.start.columnIndex, selection.end.columnIndex);
      endCol = Math.max(selection.start.columnIndex, selection.end.columnIndex);
    } else {
      startRow = endRow = focusedCell.rowIndex;
      startCol = endCol = focusedCell.columnIndex;
    }

    const rows = endRow - startRow + 1;
    const columns = endCol - startCol + 1;

    // Create placeholder data - in real implementation, this would come from the table
    const data: string[][] = Array(rows).fill(0).map(() =>
      Array(columns).fill('').map(() => 'cell_data')
    );

    const clipboardData: ClipboardData = { rows, columns, data };

    await copyToClipboard(clipboardData);
    onCopy?.(clipboardData);
  }, [disabled, navigationState, onCopy]);

  const handlePaste = useCallback(async () => {
    if (disabled || !navigationState.focusedCell) {
      return;
    }

    const data = await pasteFromClipboard();
    if (data) {
      const { rowIndex, columnIndex } = navigationState.focusedCell;
      onPaste?.(data, rowIndex, columnIndex);
    }
  }, [disabled, navigationState.focusedCell, onPaste]);

  const handleDelete = useCallback(() => {
    if (disabled || !navigationState.focusedCell || navigationState.isEditing) {
      return;
    }

    const { rowIndex, columnIndex } = navigationState.focusedCell;
    onDelete?.(rowIndex, columnIndex);
  }, [disabled, navigationState, onDelete]);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (disabled || isComposingRef.current) {
      return;
    }

    const target = event.target as HTMLElement | null;
    if (target) {
      const tagName = target.tagName;
      if (
        target.closest('[data-cell-editor="true"]') ||
        tagName === 'INPUT' ||
        tagName === 'TEXTAREA' ||
        tagName === 'SELECT' ||
        tagName === 'BUTTON' ||
        target.getAttribute('contenteditable') === 'true'
      ) {
        return;
      }
    }

    const { focusedCell, isEditing, selection } = navigationState;

    if (!focusedCell) {
      return;
    }

    const { rowIndex, columnIndex } = focusedCell;
    const { key, ctrlKey, metaKey, shiftKey, altKey } = event;
    const isModifier = ctrlKey || metaKey;

    // Handle editing state
    if (isEditing) {
      switch (key) {
        case 'Escape':
          event.preventDefault();
          stopEditing();
          break;
        case 'Enter':
        case 'Tab':
          event.preventDefault();
          stopEditing();
          // Move to next cell
          if (key === 'Enter') {
            focusCell(rowIndex + 1, columnIndex);
          } else {
            focusCell(rowIndex, columnIndex + 1);
          }
          break;
      }
      return;
    }

    // Handle non-editing state
    switch (key) {
      case 'ArrowUp':
        event.preventDefault();
        if (shiftKey && !selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              start: { rowIndex, columnIndex },
              end: { rowIndex: Math.max(0, rowIndex - 1), columnIndex },
            },
          }));
        } else if (shiftKey && selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              ...selection,
              end: { rowIndex: Math.max(0, rowIndex - 1), columnIndex },
            },
          }));
        } else {
          focusCell(rowIndex - 1, columnIndex);
        }
        break;

      case 'ArrowDown':
        event.preventDefault();
        if (shiftKey && !selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              start: { rowIndex, columnIndex },
              end: { rowIndex: Math.min(rowCount - 1, rowIndex + 1), columnIndex },
            },
          }));
        } else if (shiftKey && selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              ...selection,
              end: { rowIndex: Math.min(rowCount - 1, rowIndex + 1), columnIndex },
            },
          }));
        } else {
          focusCell(rowIndex + 1, columnIndex);
        }
        break;

      case 'ArrowLeft':
        event.preventDefault();
        if (shiftKey && !selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              start: { rowIndex, columnIndex },
              end: { rowIndex, columnIndex: Math.max(0, columnIndex - 1) },
            },
          }));
        } else if (shiftKey && selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              ...selection,
              end: { rowIndex, columnIndex: Math.max(0, columnIndex - 1) },
            },
          }));
        } else {
          focusCell(rowIndex, columnIndex - 1);
        }
        break;

      case 'ArrowRight':
        event.preventDefault();
        if (shiftKey && !selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              start: { rowIndex, columnIndex },
              end: { rowIndex, columnIndex: Math.min(columnCount - 1, columnIndex + 1) },
            },
          }));
        } else if (shiftKey && selection) {
          setNavigationState(prev => ({
            ...prev,
            selection: {
              ...selection,
              end: { rowIndex, columnIndex: Math.min(columnCount - 1, columnIndex + 1) },
            },
          }));
        } else {
          focusCell(rowIndex, columnIndex + 1);
        }
        break;

      case 'Tab':
        event.preventDefault();
        if (shiftKey) {
          focusCell(rowIndex, columnIndex - 1);
        } else {
          focusCell(rowIndex, columnIndex + 1);
        }
        break;

      case 'Enter':
        event.preventDefault();
        if (shiftKey) {
          focusCell(rowIndex - 1, columnIndex);
        } else {
          startEditing();
        }
        break;

      case 'F2':
        event.preventDefault();
        startEditing();
        break;

      case 'Escape':
        event.preventDefault();
        setNavigationState(prev => ({
          ...prev,
          selection: null,
        }));
        break;

      case 'Delete':
      case 'Backspace':
        event.preventDefault();
        handleDelete();
        break;

      case 'c':
        if (isModifier) {
          event.preventDefault();
          handleCopy();
        }
        break;

      case 'v':
        if (isModifier) {
          event.preventDefault();
          handlePaste();
        }
        break;

      case 'z':
        if (isModifier && !shiftKey) {
          event.preventDefault();
          onUndo?.();
        } else if (isModifier && shiftKey) {
          event.preventDefault();
          onRedo?.();
        }
        break;

      case 'y':
        if (isModifier) {
          event.preventDefault();
          onRedo?.();
        }
        break;

      case 'a':
        if (isModifier) {
          event.preventDefault();
          setNavigationState(prev => ({
            ...prev,
            selection: {
              start: { rowIndex: 0, columnIndex: 0 },
              end: { rowIndex: rowCount - 1, columnIndex: columnCount - 1 },
            },
          }));
        }
        break;

      case 'Home':
        event.preventDefault();
        if (isModifier) {
          focusCell(0, 0);
        } else {
          focusCell(rowIndex, 0);
        }
        break;

      case 'End':
        event.preventDefault();
        if (isModifier) {
          focusCell(rowCount - 1, columnCount - 1);
        } else {
          focusCell(rowIndex, columnCount - 1);
        }
        break;

      case 'PageUp':
        event.preventDefault();
        focusCell(Math.max(0, rowIndex - 10), columnIndex);
        break;

      case 'PageDown':
        event.preventDefault();
        focusCell(Math.min(rowCount - 1, rowIndex + 10), columnIndex);
        break;

      default:
        // Start editing if a printable character is typed
        if (key.length === 1 && !isModifier && !altKey) {
          startEditing();
        }
        break;
    }
  }, [
    disabled,
    navigationState,
    rowCount,
    columnCount,
    focusCell,
    startEditing,
    stopEditing,
    handleCopy,
    handlePaste,
    handleDelete,
    onUndo,
    onRedo,
  ]);

  const handleCompositionStart = useCallback(() => {
    isComposingRef.current = true;
  }, []);

  const handleCompositionEnd = useCallback(() => {
    isComposingRef.current = false;
  }, []);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    container.addEventListener('keydown', handleKeyDown);
    container.addEventListener('compositionstart', handleCompositionStart);
    container.addEventListener('compositionend', handleCompositionEnd);

    return () => {
      container.removeEventListener('keydown', handleKeyDown);
      container.removeEventListener('compositionstart', handleCompositionStart);
      container.removeEventListener('compositionend', handleCompositionEnd);
    };
  }, [handleKeyDown, handleCompositionStart, handleCompositionEnd]);

  return {
    navigationState,
    containerRef,
    focusCell,
    startEditing,
    stopEditing,
    setNavigationState,
  };
};
