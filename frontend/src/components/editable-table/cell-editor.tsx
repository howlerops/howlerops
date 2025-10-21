import React, { useState, useEffect, useRef, useCallback } from 'react';
import { cn } from '../../utils/cn';
import type { CellEditorProps } from '../../types/table';
import { validateCellValue, parseCellValue, isEqual } from '../../utils/table';

export const CellEditor: React.FC<CellEditorProps> = ({
  value,
  type,
  onChange,
  onCancel,
  onSave,
  validation,
  options = [],
  required = false,
  autoFocus = true,
  className,
}) => {
  const [localValue, setLocalValue] = useState<string>(
    value?.toString() ?? ''
  );
  const [isValid, setIsValid] = useState(true);
  const [error, setError] = useState<string>();

  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(null);
  const onChangeRef = useRef(onChange);

  // Keep onChange ref up to date
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  // Focus the input when the editor is mounted
  useEffect(() => {
    if (autoFocus && inputRef.current) {
      inputRef.current.focus();
      if ('select' in inputRef.current) {
        inputRef.current.select();
      }
    }
  }, [autoFocus]);

  // Validate the value whenever it changes
  useEffect(() => {
    const parsedValue = parseCellValue(localValue, type);
    const validationResult = validateCellValue(parsedValue, {
      id: '',
      accessorKey: '',
      header: '',
      type,
      validation,
      required,
      options,
    });

    setIsValid(validationResult.isValid);
    setError(validationResult.error);
    
    // Always call onChange with validation state
    // The parent should handle this properly to avoid infinite loops
    onChangeRef.current(parsedValue, validationResult.isValid, validationResult.error);
  }, [localValue, type, validation, required, options]);

  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    switch (event.key) {
      case 'Enter':
        event.preventDefault();
        if (isValid) {
          onSave();
        }
        break;
      case 'Escape':
        event.preventDefault();
        onCancel();
        break;
      case 'Tab':
        event.preventDefault();
        if (isValid) {
          onSave();
        }
        break;
    }
  }, [isValid, onSave, onCancel]);

  const handleBlur = useCallback(() => {
    if (isValid) {
      onSave();
    } else {
      onCancel();
    }
  }, [isValid, onSave, onCancel]);

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
        'border-destructive focus:ring-red-500': !isValid,
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
            {options.map(option => (
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
    <div className="relative w-full h-full">
      {renderInput()}

      {/* Validation error */}
      {error && (
        <div className="absolute z-50 mt-1 p-2 bg-destructive border border-destructive rounded text-xs text-destructive shadow-lg min-w-max">
          {error}
        </div>
      )}

      {/* Save/Cancel buttons for complex editors */}
      {(type === 'text' && localValue.length > 100) && (
        <div className="absolute z-40 bottom-0 right-0 flex gap-1 p-1 bg-white border border-gray-300 rounded shadow">
          <button
            onClick={onSave}
            disabled={!isValid}
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