import React, { useCallback, useEffect, useMemo,useRef, useState } from 'react';

import type { CellEditorProps } from '../../types/table';
import { cn } from '../../utils/cn';
import { parseCellValue,validateCellValue } from '../../utils/table';

const EMPTY_OPTIONS: string[] = [];

const toDateInputValue = (raw: unknown): string => {
  if (!raw) return '';
  const date = new Date(String(raw));
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString().slice(0, 10);
};

const toDateTimeInputValue = (raw: unknown): string => {
  if (!raw) return '';
  const date = new Date(String(raw));
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString().slice(0, 16);
};

export const CellEditor: React.FC<CellEditorProps> = ({
  value,
  type,
  onChange,
  onCancel,
  onSave,
  validation,
  options,
  required = false,
  autoFocus = true,
  className,
}) => {
  const editorOptions = options ?? EMPTY_OPTIONS;
  const deriveInitialValue = useCallback(() => {
    if (type === 'date') {
      return toDateInputValue(value);
    }
    if (type === 'datetime') {
      return toDateTimeInputValue(value);
    }
    return value?.toString() ?? '';
  }, [type, value]);

  const [localValue, setLocalValue] = useState<string>(deriveInitialValue);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(null);
  const hasFocusedRef = useRef(false);
  const onChangeRef = useRef(onChange);

  // Keep onChange ref up to date
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    setLocalValue(deriveInitialValue());
  }, [deriveInitialValue]);

  // Focus the input when the editor is mounted
  useEffect(() => {
    if (!autoFocus || !inputRef.current || hasFocusedRef.current) return;

    hasFocusedRef.current = true;
    const element = inputRef.current;
    element.focus();

    if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
      const length = element.value.length;
      try {
        element.setSelectionRange(length, length);
      } catch {
        // Ignore browsers that don't support setSelectionRange (e.g. type="number")
      }
    }
  }, [autoFocus]);

  const validationResult = useMemo(() => {
    const parsedValue = parseCellValue(localValue, type);
    const result = validateCellValue(parsedValue, {
      id: '',
      accessorKey: '',
      header: '',
      type,
      validation,
      required,
      options: editorOptions,
    });

    return {
      parsedValue,
      isValid: result.isValid,
      error: result.error,
    };
  }, [localValue, type, validation, required, editorOptions]);

  // Notify parent about validation changes
  useEffect(() => {
    onChangeRef.current(
      validationResult.parsedValue,
      validationResult.isValid,
      validationResult.error
    );
  }, [validationResult]);

  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    switch (event.key) {
      case 'Enter':
        event.preventDefault();
        if (validationResult.isValid) {
          onSave();
        }
        break;
      case 'Escape':
        event.preventDefault();
        onCancel();
        break;
      case 'Tab':
        event.preventDefault();
        if (validationResult.isValid) {
          onSave();
        }
        break;
    }
  }, [validationResult.isValid, onSave, onCancel]);

  const handleBlur = useCallback(() => {
    if (validationResult.isValid) {
      onSave();
    } else {
      onCancel();
    }
  }, [validationResult.isValid, onSave, onCancel]);

  const handleChange = useCallback((
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    setLocalValue(event.target.value);
  }, []);

  const renderInput = () => {
    const baseClassName = cn(
      'w-full h-full px-2 py-1 text-sm border-0 outline-none resize-none',
      'focus:ring-2 focus:ring-blue-500 focus:ring-inset',
      {
        'border-destructive focus:ring-red-500': !validationResult.isValid,
      },
      className
    );

    switch (type) {
      case 'boolean':
        return (
          <select
            ref={inputRef as React.RefObject<HTMLSelectElement>}
            value={localValue}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={baseClassName}
          >
            <option value="">Select...</option>
            <option value="true">Yes</option>
            <option value="false">No</option>
          </select>
        );

      case 'select':
        return (
          <select
            ref={inputRef as React.RefObject<HTMLSelectElement>}
            value={localValue}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={baseClassName}
          >
            {!required && <option value="">Select...</option>}
            {editorOptions.map(option => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        );

      case 'number':
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="number"
            value={localValue}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={cn(baseClassName, 'text-right font-mono')}
            min={validation?.min}
            max={validation?.max}
            step="any"
          />
        );

      case 'date':
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="date"
            value={localValue}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={cn(baseClassName, 'font-mono')}
          />
        );
      case 'datetime':
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="datetime-local"
            value={localValue}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={cn(baseClassName, 'font-mono')}
          />
        );

      case 'text':
      default: {
        // Check if it's a long text that might need a textarea
        const isLongText = localValue.length > 100 || localValue.includes('\n');

        if (isLongText) {
          return (
            <textarea
              ref={inputRef as React.RefObject<HTMLTextAreaElement>}
              value={localValue}
              onChange={handleChange}
              onKeyDown={handleKeyDown}
              onBlur={handleBlur}
              className={cn(baseClassName, 'min-h-[60px]')}
              rows={3}
              maxLength={validation?.max}
            />
          );
        }

        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="text"
            value={localValue}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={baseClassName}
            maxLength={validation?.max}
            pattern={validation?.pattern?.source}
          />
        );
      }
    }
  };

  return (
    <div className="relative w-full h-full" data-cell-editor="true">
      {renderInput()}

      {/* Validation error */}
      {validationResult.error && (
        <div className="absolute z-50 mt-1 p-2 bg-destructive border border-destructive rounded text-xs text-destructive shadow-lg min-w-max">
          {validationResult.error}
        </div>
      )}

      {/* Save/Cancel buttons for complex editors */}
      {(type === 'text' && localValue.length > 100) && (
        <div className="absolute z-40 bottom-0 right-0 flex gap-1 p-1 bg-white border border-gray-300 rounded shadow">
          <button
            onClick={onSave}
            disabled={!validationResult.isValid}
            className="px-2 py-1 text-xs bg-primary text-white rounded disabled:opacity-50 hover:bg-primary"
            type="button"
          >
            Save
          </button>
          <button
            onClick={onCancel}
            className="px-2 py-1 text-xs bg-gray-500 text-white rounded hover:bg-gray-600"
            type="button"
          >
            Cancel
          </button>
        </div>
      )}
    </div>
  );
};
