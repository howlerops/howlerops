/**
 * .env Import State Reducer
 *
 * Manages state transitions for the .env import dialog flow.
 *
 * @module lib/export-import/env-parser/reducer
 */

import type {
  EnvConnectionExtractionResult,
  EnvImportAction,
  EnvImportDialogState,
  EnvImportOptions,
  EnvParseResult,
  ParsedEnvConnection,
} from './types'
import { INITIAL_ENV_IMPORT_STATE } from './types'

/**
 * Reducer for .env import dialog state
 *
 * @param state - Current state
 * @param action - Action to apply
 * @returns New state
 */
export function envImportReducer(
  state: EnvImportDialogState,
  action: EnvImportAction
): EnvImportDialogState {
  switch (action.type) {
    case 'SET_FILE':
      return {
        ...state,
        file: action.file,
        step: 'parsing',
        error: null,
      }

    case 'SET_PARSING':
      return {
        ...state,
        step: 'parsing',
        error: null,
      }

    case 'SET_PARSE_RESULT':
      return {
        ...state,
        envParseResult: action.result,
      }

    case 'SET_EXTRACTION_RESULT':
      return {
        ...state,
        extractionResult: action.result,
        step: action.result.connections.length > 0 ? 'preview' : 'error',
        error:
          action.result.connections.length === 0
            ? 'No database connections found in this file'
            : null,
      }

    case 'SET_STEP':
      return {
        ...state,
        step: action.step,
        error: null,
      }

    case 'START_EDITING':
      return {
        ...state,
        editingConnectionId: action.connectionId,
        step: 'editing',
      }

    case 'STOP_EDITING':
      return {
        ...state,
        editingConnectionId: null,
        step: 'preview',
      }

    case 'UPDATE_CONNECTION':
      if (!state.extractionResult) return state
      return {
        ...state,
        extractionResult: {
          ...state.extractionResult,
          connections: state.extractionResult.connections.map(conn =>
            conn.tempId === action.connectionId
              ? { ...conn, ...action.updates, isReviewed: true }
              : conn
          ),
        },
      }

    case 'TOGGLE_SKIP':
      if (!state.extractionResult) return state
      return {
        ...state,
        extractionResult: {
          ...state.extractionResult,
          connections: state.extractionResult.connections.map(conn =>
            conn.tempId === action.connectionId
              ? { ...conn, isSkipped: !conn.isSkipped }
              : conn
          ),
        },
      }

    case 'REMOVE_CONNECTION':
      if (!state.extractionResult) return state
      return {
        ...state,
        extractionResult: {
          ...state.extractionResult,
          connections: state.extractionResult.connections.filter(
            conn => conn.tempId !== action.connectionId
          ),
        },
      }

    case 'SET_OPTIONS':
      return {
        ...state,
        importOptions: {
          ...state.importOptions,
          ...action.options,
        },
      }

    case 'SET_IMPORT_RESULT':
      return {
        ...state,
        importResult: action.result,
        step: 'complete',
      }

    case 'SET_ERROR':
      return {
        ...state,
        error: action.error,
        step: 'error',
      }

    case 'RESET':
      return INITIAL_ENV_IMPORT_STATE

    default:
      return state
  }
}

/**
 * Action creators for type safety
 */
export const envImportActions = {
  setFile: (file: File): EnvImportAction => ({
    type: 'SET_FILE',
    file,
  }),

  setParsing: (): EnvImportAction => ({
    type: 'SET_PARSING',
  }),

  setParseResult: (result: EnvParseResult): EnvImportAction => ({
    type: 'SET_PARSE_RESULT',
    result,
  }),

  setExtractionResult: (
    result: EnvConnectionExtractionResult
  ): EnvImportAction => ({
    type: 'SET_EXTRACTION_RESULT',
    result,
  }),

  setStep: (
    step: EnvImportDialogState['step']
  ): EnvImportAction => ({
    type: 'SET_STEP',
    step,
  }),

  startEditing: (connectionId: string): EnvImportAction => ({
    type: 'START_EDITING',
    connectionId,
  }),

  stopEditing: (): EnvImportAction => ({
    type: 'STOP_EDITING',
  }),

  updateConnection: (
    connectionId: string,
    updates: Partial<ParsedEnvConnection>
  ): EnvImportAction => ({
    type: 'UPDATE_CONNECTION',
    connectionId,
    updates,
  }),

  toggleSkip: (connectionId: string): EnvImportAction => ({
    type: 'TOGGLE_SKIP',
    connectionId,
  }),

  removeConnection: (connectionId: string): EnvImportAction => ({
    type: 'REMOVE_CONNECTION',
    connectionId,
  }),

  setOptions: (options: Partial<EnvImportOptions>): EnvImportAction => ({
    type: 'SET_OPTIONS',
    options,
  }),

  setImportResult: (
    result: EnvImportDialogState['importResult']
  ): EnvImportAction => ({
    type: 'SET_IMPORT_RESULT',
    result: result!,
  }),

  setError: (error: string): EnvImportAction => ({
    type: 'SET_ERROR',
    error,
  }),

  reset: (): EnvImportAction => ({
    type: 'RESET',
  }),
}

/**
 * Selectors for accessing state
 */
export const envImportSelectors = {
  /** Get connections that will be imported (non-skipped) */
  getConnectionsToImport: (
    state: EnvImportDialogState
  ): ParsedEnvConnection[] => {
    if (!state.extractionResult) return []
    return state.extractionResult.connections.filter(c => !c.isSkipped)
  },

  /** Get the connection currently being edited */
  getEditingConnection: (
    state: EnvImportDialogState
  ): ParsedEnvConnection | undefined => {
    if (!state.extractionResult || !state.editingConnectionId) return undefined
    return state.extractionResult.connections.find(
      c => c.tempId === state.editingConnectionId
    )
  },

  /** Check if any connections have validation errors */
  hasValidationErrors: (state: EnvImportDialogState): boolean => {
    if (!state.extractionResult) return false
    return state.extractionResult.connections.some(
      c => !c.isSkipped && c.validationErrors.length > 0
    )
  },

  /** Get count of connections by status */
  getConnectionCounts: (
    state: EnvImportDialogState
  ): {
    total: number
    toImport: number
    skipped: number
    invalid: number
  } => {
    if (!state.extractionResult) {
      return { total: 0, toImport: 0, skipped: 0, invalid: 0 }
    }

    const connections = state.extractionResult.connections
    return {
      total: connections.length,
      toImport: connections.filter(c => !c.isSkipped).length,
      skipped: connections.filter(c => c.isSkipped).length,
      invalid: connections.filter(
        c => !c.isSkipped && c.validationErrors.length > 0
      ).length,
    }
  },

  /** Check if import can proceed */
  canProceedToImport: (state: EnvImportDialogState): boolean => {
    if (!state.extractionResult) return false

    const toImport = state.extractionResult.connections.filter(c => !c.isSkipped)
    if (toImport.length === 0) return false

    // All to-import connections must have no validation errors
    return toImport.every(c => c.validationErrors.length === 0)
  },
}
