/**
 * Query Execution Store Compatibility Shim
 *
 * The execution runtime now lives in query-engine-store.ts.
 * This file preserves the older public surface while forwarding all consumers
 * to the unified query engine state layer.
 */

export {
  useExecutingQueries,
  useIsExecuting,
  useQueryEngineActions,
  useQueryEngineStore,
  useQueryEngineActions as useQueryExecutionActions,
  useQueryEngineStore as useQueryExecutionStore,
} from './query-engine-store'
