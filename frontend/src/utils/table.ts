import { CellValue, ClipboardData, ExportOptions,TableColumn, TableRow, ValidationResult } from '../types/table';

export const validateCellValue = (
  value: CellValue,
  column: TableColumn
): ValidationResult => {
  if (column.required && (value === null || value === undefined || value === '')) {
    return { isValid: false, error: 'This field is required' };
  }

  if (value === null || value === undefined || value === '') {
    return { isValid: true };
  }

  const { validation, type } = column;

  // Type-specific validation
  switch (type) {
    case 'number': {
      const numValue = Number(value);
      if (isNaN(numValue)) {
        return { isValid: false, error: 'Must be a valid number' };
      }
      if (validation?.min !== undefined && numValue < validation.min) {
        return { isValid: false, error: `Minimum value is ${validation.min}` };
      }
      if (validation?.max !== undefined && numValue > validation.max) {
        return { isValid: false, error: `Maximum value is ${validation.max}` };
      }
      break;
    }

    case 'text': {
      const strValue = String(value);
      if (validation?.pattern && !validation.pattern.test(strValue)) {
        return {
          isValid: false,
          error: validation.message || 'Invalid format'
        };
      }
      if (validation?.min !== undefined && strValue.length < validation.min) {
        return {
          isValid: false,
          error: `Minimum length is ${validation.min} characters`
        };
      }
      if (validation?.max !== undefined && strValue.length > validation.max) {
        return {
          isValid: false,
          error: `Maximum length is ${validation.max} characters`
        };
      }
      break;
    }

    case 'select':
      if (column.options && !column.options.includes(String(value))) {
        return { isValid: false, error: 'Invalid selection' };
      }
      break;

    case 'date':
    case 'datetime': {
      const dateValue = new Date(String(value));
      if (isNaN(dateValue.getTime())) {
        return { isValid: false, error: 'Must be a valid date' };
      }
      break;
    }
  }

  return { isValid: true };
};

export const formatCellValue = (value: CellValue, type: TableColumn['type']): string => {
  if (value === null || value === undefined) {
    return '';
  }

  switch (type) {
    case 'number':
      return typeof value === 'number' ? value.toLocaleString() : String(value);
    case 'boolean':
      return value ? 'Yes' : 'No';
    case 'date': {
      const date = new Date(String(value));
      return isNaN(date.getTime()) ? String(value) : date.toLocaleDateString();
    }
    case 'datetime': {
      const date = new Date(String(value));
      return isNaN(date.getTime()) ? String(value) : date.toLocaleString();
    }
    default:
      return String(value);
  }
};

export const parseCellValue = (value: string, type: TableColumn['type']): CellValue => {
  if (value === '') {
    return null;
  }

  switch (type) {
    case 'number': {
      const num = Number(value);
      return isNaN(num) ? value : num;
    }
    case 'boolean':
      return value.toLowerCase() === 'true' || value === '1' || value.toLowerCase() === 'yes';
    case 'date': {
      const date = new Date(value);
      return isNaN(date.getTime()) ? value : date.toISOString();
    }
    case 'datetime': {
      const date = new Date(value);
      return isNaN(date.getTime()) ? value : date.toISOString();
    }
    default:
      return value;
  }
};

export const copyToClipboard = async (data: ClipboardData): Promise<boolean> => {
  try {
    const text = data.data
      .map(row => row.map(cell => String(cell || '')).join('\t'))
      .join('\n');

    await navigator.clipboard.writeText(text);
    return true;
  } catch (error) {
    console.error('Failed to copy to clipboard:', error);
    return false;
  }
};

export const pasteFromClipboard = async (): Promise<ClipboardData | null> => {
  try {
    const text = await navigator.clipboard.readText();
    const rows = text.split('\n').filter(row => row.trim());
    const data = rows.map(row => row.split('\t'));

    return {
      rows: data.length,
      columns: data[0]?.length || 0,
      data
    };
  } catch (error) {
    console.error('Failed to paste from clipboard:', error);
    return null;
  }
};

export const exportData = (
  data: TableRow[],
  columns: TableColumn[],
  options: ExportOptions
): void => {
  const { format, filename = 'data', selectedOnly: _selectedOnly = false, includeHeaders = true } = options;

  let content = '';
  const headers = columns.map(col => col.header);
  const exportData = data;

  switch (format) {
    case 'csv': {
      const csvRows = [];
      if (includeHeaders) {
        csvRows.push(headers.join(','));
      }
      exportData.forEach(row => {
        const values = columns.map(col => {
          const value = col.accessorKey ? row[col.accessorKey] : undefined;
          const formatted = formatCellValue(value, col.type);
          return `"${formatted.replace(/"/g, '""')}"`;
        });
        csvRows.push(values.join(','));
      });
      content = csvRows.join('\n');
      break;
    }

    case 'json':
      content = JSON.stringify(exportData, null, 2);
      break;

    default:
      throw new Error(`Unsupported export format: ${format}`);
  }

  const blob = new Blob([content], {
    type: format === 'json' ? 'application/json' : 'text/csv'
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `${filename}.${format}`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};

export const debounce = <T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void => {
  let timeout: NodeJS.Timeout;

  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
};

export const throttle = <T extends (...args: unknown[]) => unknown>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void => {
  let inThrottle: boolean;

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
};

export const getColumnWidth = (
  column: TableColumn,
  data: TableRow[],
  minWidth = 80,
  maxWidth = 400
): number => {
  if (column.width) {
    return column.width;
  }

  const resolvedMin = column.minWidth ?? (
    column.type === 'boolean' ? 110 :
    column.type === 'number' ? 140 :
    minWidth
  );

  const resolvedMax = Math.max(
    resolvedMin,
    column.maxWidth ?? (
      column.longText ? 700 :
      column.type === 'number' ? 280 :
      maxWidth
    )
  );

  const headerWidth = column.header.length * 8 + 48;
  const samples = data.slice(0, 200);
  const charWidth = column.monospace ? 8.5 : 7;

  let longest = 0;
  if (samples.length > 0 && column.accessorKey) {
    for (const row of samples) {
      const formatted = formatCellValue(row[column.accessorKey], column.type);
      longest = Math.max(longest, formatted.length);
    }
  }

  const contentWidth = Math.max(
    column.preferredWidth ?? 0,
    longest > 0 ? longest * charWidth + 32 : 0
  );

  const calculatedWidth = Math.max(headerWidth, contentWidth, resolvedMin);
  return Math.min(calculatedWidth, resolvedMax);
};

export const generateTableId = (): string => {
  return `table_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

export const isEqual = (a: unknown, b: unknown): boolean => {
  return JSON.stringify(a) === JSON.stringify(b);
};

export const cloneDeep = <T>(obj: T): T => {
  return JSON.parse(JSON.stringify(obj));
};

export const requestAnimationFrame = (callback: () => void): number => {
  if (typeof window !== 'undefined' && window.requestAnimationFrame) {
    return window.requestAnimationFrame(callback);
  }
  return setTimeout(callback, 16) as unknown as number; // 60fps fallback
};

export const cancelAnimationFrame = (id: number): void => {
  if (typeof window !== 'undefined' && window.cancelAnimationFrame) {
    window.cancelAnimationFrame(id);
  } else {
    clearTimeout(id);
  }
};
