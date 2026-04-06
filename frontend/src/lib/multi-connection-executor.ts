/**
 * Compatibility shim for the old multi-connection executor import path.
 * The actual implementation now lives under lib/query-engine/.
 */

export type {
  MergedResult,
  MultiConnectionResult,
} from './query-engine/multi-connection-engine'
export {
  createMultiConnectionQueryEngine as createMultiConnectionExecutor,
  MultiConnectionQueryEngine as MultiConnectionExecutor,
} from './query-engine/multi-connection-engine'
