/**
 * IndexedDB Client Wrapper
 *
 * Type-safe wrapper around IndexedDB with:
 * - Async/await interface
 * - Transaction management
 * - Connection pooling
 * - Error handling and retries
 * - Schema migrations
 *
 * @module lib/storage/indexeddb-client
 */

import {
  type PaginatedResult,
  type QueryOptions,
  QuotaExceededError,
  StorageError,
  type StoreName,
  TransactionError,
  VersionMismatchError,
} from '@/types/storage'

import {
  CURRENT_VERSION,
  DB_NAME,
  getCurrentSchema,
  getSchemaVersion,
} from './schema'

/**
 * Transaction mode types
 */
export type TransactionMode = 'readonly' | 'readwrite'

/**
 * IndexedDB client for managing database operations
 */
export class IndexedDBClient {
  private db: IDBDatabase | null = null
  private opening: Promise<IDBDatabase> | null = null
  private readonly maxRetries = 3
  private readonly retryDelay = 100

  /**
   * Get or create database connection
   */
  private async getDB(): Promise<IDBDatabase> {
    if (this.db && this.db.version === CURRENT_VERSION) {
      return this.db
    }

    // If already opening, wait for it
    if (this.opening) {
      return this.opening
    }

    // Start opening process
    this.opening = this.open()
    try {
      this.db = await this.opening
      return this.db
    } finally {
      this.opening = null
    }
  }

  /**
   * Open database connection with schema migration
   */
  private async open(): Promise<IDBDatabase> {
    return new Promise<IDBDatabase>((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, CURRENT_VERSION)

      request.onerror = () => {
        reject(
          new StorageError(
            `Failed to open database: ${request.error?.message}`,
            'DB_OPEN_ERROR',
            request.error ?? undefined
          )
        )
      }

      request.onsuccess = () => {
        const db = request.result

        // Handle unexpected close
        db.onversionchange = () => {
          db.close()
          this.db = null
          console.warn('Database version changed, connection closed')
        }

        resolve(db)
      }

      request.onupgradeneeded = (event) => {
        const db = request.result
        const transaction = request.transaction!
        const oldVersion = event.oldVersion
        const newVersion = event.newVersion ?? CURRENT_VERSION

        console.log(`Upgrading database from v${oldVersion} to v${newVersion}`)

        try {
          // If this is a new database, create all stores
          if (oldVersion === 0) {
            const schema = getCurrentSchema()
            this.createStores(db, schema.stores)

            // Run initial migration
            if (schema.migrate) {
              schema.migrate(db, transaction)
            }
          } else {
            // Run migrations for each version
            for (let v = oldVersion + 1; v <= newVersion; v++) {
              const schema = getSchemaVersion(v)
              if (!schema) {
                throw new VersionMismatchError(`Schema version ${v} not found`)
              }

              // Create new stores
              this.createStores(db, schema.stores)

              // Run migration
              if (schema.migrate) {
                schema.migrate(db, transaction)
              }
            }
          }
        } catch (error) {
          transaction.abort()
          reject(
            error instanceof StorageError
              ? error
              : new StorageError(
                  `Migration failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
                  'MIGRATION_ERROR',
                  error instanceof Error ? error : undefined
                )
          )
        }
      }

      request.onblocked = () => {
        console.warn('Database upgrade blocked by other open connections')
      }
    })
  }

  /**
   * Create object stores from configuration
   */
  private createStores(db: IDBDatabase, stores: Array<{ name: string; keyPath: string; autoIncrement?: boolean; indexes?: Array<{ name: string; keyPath: string | string[]; unique?: boolean; multiEntry?: boolean }> }>): void {
    stores.forEach((storeConfig) => {
      // Skip if store already exists
      if (db.objectStoreNames.contains(storeConfig.name)) {
        return
      }

      const store = db.createObjectStore(storeConfig.name, {
        keyPath: storeConfig.keyPath,
        autoIncrement: storeConfig.autoIncrement ?? false,
      })

      // Create indexes
      storeConfig.indexes?.forEach((indexConfig) => {
        store.createIndex(indexConfig.name, indexConfig.keyPath, {
          unique: indexConfig.unique ?? false,
          multiEntry: indexConfig.multiEntry ?? false,
        })
      })

      console.log(`Created object store: ${storeConfig.name}`)
    })
  }

  /**
   * Execute operation with automatic retry on failure
   */
  private async withRetry<T>(
    operation: () => Promise<T>,
    retries = this.maxRetries
  ): Promise<T> {
    try {
      return await operation()
    } catch (error) {
      if (retries > 0) {
        await new Promise((resolve) => setTimeout(resolve, this.retryDelay))
        return this.withRetry(operation, retries - 1)
      }
      throw error
    }
  }

  /**
   * Get object store for operation
   */
  private getStore(
    storeName: StoreName,
    mode: TransactionMode = 'readonly'
  ): Promise<IDBObjectStore> {
    return this.withRetry(async () => {
      const db = await this.getDB()
      const transaction = db.transaction(storeName, mode)
      return transaction.objectStore(storeName)
    })
  }

  /**
   * Execute multiple operations in a single transaction
   */
  async transaction<T>(
    storeNames: StoreName | StoreName[],
    mode: TransactionMode,
    callback: (stores: IDBObjectStore | IDBObjectStore[]) => Promise<T>
  ): Promise<T> {
    const db = await this.getDB()
    const names = Array.isArray(storeNames) ? storeNames : [storeNames]
    const transaction = db.transaction(names, mode)

    return new Promise<T>((resolve, reject) => {
      transaction.onerror = () => {
        reject(
          new TransactionError(
            `Transaction failed: ${transaction.error?.message}`,
            transaction.error ?? undefined
          )
        )
      }

      transaction.onabort = () => {
        reject(
          new TransactionError(
            'Transaction aborted',
            transaction.error ?? undefined
          )
        )
      }

      // Get stores
      const stores = Array.isArray(storeNames)
        ? names.map((name) => transaction.objectStore(name))
        : transaction.objectStore(storeNames)

      // Execute callback
      callback(stores)
        .then(resolve)
        .catch((error) => {
          transaction.abort()
          reject(error)
        })
    })
  }

  /**
   * Get a single record by primary key
   */
  async get<T>(storeName: StoreName, key: IDBValidKey): Promise<T | null> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readonly')
      return new Promise<T | null>((resolve, reject) => {
        const request = store.get(key)

        request.onsuccess = () => {
          resolve(request.result ?? null)
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to get record: ${request.error?.message}`,
              'GET_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Get all records from a store
   */
  async getAll<T>(
    storeName: StoreName,
    options?: QueryOptions
  ): Promise<T[]> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readonly')
      const source = options?.index ? store.index(options.index) : store

      return new Promise<T[]>((resolve, reject) => {
        const request = source.getAll(options?.range, options?.limit)

        request.onsuccess = () => {
          let results = request.result ?? []

          // Apply offset if specified
          if (options?.offset) {
            results = results.slice(options.offset)
          }

          resolve(results)
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to get all records: ${request.error?.message}`,
              'GETALL_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Get paginated records using cursor
   */
  async getPaginated<T>(
    storeName: StoreName,
    options?: QueryOptions
  ): Promise<PaginatedResult<T>> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readonly')
      const source = options?.index ? store.index(options.index) : store

      return new Promise<PaginatedResult<T>>((resolve, reject) => {
        const items: T[] = []
        const limit = options?.limit ?? 50
        const offset = options?.offset ?? 0
        let skipped = 0
        let count = 0

        const request = source.openCursor(
          options?.range,
          options?.direction ?? 'next'
        )

        request.onsuccess = () => {
          const cursor = request.result

          if (!cursor) {
            // No more results
            resolve({
              items,
              hasMore: false,
              nextCursor: undefined,
            })
            return
          }

          // Skip offset records
          if (skipped < offset) {
            skipped++
            cursor.continue()
            return
          }

          // Collect result
          items.push(cursor.value)
          count++

          // Check if we have enough results
          if (count >= limit) {
            // Try to check if there are more results
            cursor.continue()
            return
          }

          cursor.continue()
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to get paginated records: ${request.error?.message}`,
              'PAGINATED_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Count records in a store
   */
  async count(
    storeName: StoreName,
    options?: { index?: string; range?: IDBKeyRange }
  ): Promise<number> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readonly')
      const source = options?.index ? store.index(options.index) : store

      return new Promise<number>((resolve, reject) => {
        const request = source.count(options?.range)

        request.onsuccess = () => {
          resolve(request.result)
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to count records: ${request.error?.message}`,
              'COUNT_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Put (insert or update) a record
   */
  async put<T>(storeName: StoreName, record: T): Promise<IDBValidKey> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readwrite')

      return new Promise<IDBValidKey>((resolve, reject) => {
        const request = store.put(record)

        request.onsuccess = () => {
          resolve(request.result)
        }

        request.onerror = () => {
          // Check for quota exceeded
          if (request.error?.name === 'QuotaExceededError') {
            reject(new QuotaExceededError(request.error.message, request.error))
          } else {
            reject(
              new StorageError(
                `Failed to put record: ${request.error?.message}`,
                'PUT_ERROR',
                request.error ?? undefined
              )
            )
          }
        }
      })
    })
  }

  /**
   * Put multiple records in a single transaction
   */
  async putMany<T>(storeName: StoreName, records: T[]): Promise<IDBValidKey[]> {
    const db = await this.getDB()
    const transaction = db.transaction(storeName, 'readwrite')
    const store = transaction.objectStore(storeName)

      return new Promise<IDBValidKey[]>((resolve, reject) => {
        const keys: IDBValidKey[] = []

      transaction.onerror = () => {
        reject(
          new TransactionError(
            `Batch put failed: ${transaction.error?.message}`,
            transaction.error ?? undefined
          )
        )
      }

      transaction.oncomplete = () => {
        resolve(keys)
      }

      records.forEach((record) => {
        const request = store.put(record)

        request.onsuccess = () => {
          keys.push(request.result)
        }

        request.onerror = () => {
          transaction.abort()
          if (request.error?.name === 'QuotaExceededError') {
            reject(new QuotaExceededError(request.error.message, request.error))
          }
        }
      })
    })
  }

  /**
   * Delete a record by primary key
   */
  async delete(storeName: StoreName, key: IDBValidKey): Promise<void> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readwrite')

      return new Promise<void>((resolve, reject) => {
        const request = store.delete(key)

        request.onsuccess = () => {
          resolve()
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to delete record: ${request.error?.message}`,
              'DELETE_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Delete multiple records by key range
   */
  async deleteRange(
    storeName: StoreName,
    range: IDBKeyRange
  ): Promise<number> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readwrite')

      return new Promise<number>((resolve, reject) => {
        let deleted = 0
        const request = store.openCursor(range)

        request.onsuccess = () => {
          const cursor = request.result
          if (cursor) {
            cursor.delete()
            deleted++
            cursor.continue()
          } else {
            resolve(deleted)
          }
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to delete range: ${request.error?.message}`,
              'DELETE_RANGE_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Clear all records from a store
   */
  async clear(storeName: StoreName): Promise<void> {
    return this.withRetry(async () => {
      const store = await this.getStore(storeName, 'readwrite')

      return new Promise<void>((resolve, reject) => {
        const request = store.clear()

        request.onsuccess = () => {
          resolve()
        }

        request.onerror = () => {
          reject(
            new StorageError(
              `Failed to clear store: ${request.error?.message}`,
              'CLEAR_ERROR',
              request.error ?? undefined
            )
          )
        }
      })
    })
  }

  /**
   * Close database connection
   */
  close(): void {
    if (this.db) {
      this.db.close()
      this.db = null
    }
  }

  /**
   * Delete the entire database
   */
  static async deleteDatabase(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      const request = indexedDB.deleteDatabase(DB_NAME)

      request.onsuccess = () => {
        console.log('Database deleted successfully')
        resolve()
      }

      request.onerror = () => {
        reject(
          new StorageError(
            `Failed to delete database: ${request.error?.message}`,
            'DELETE_DB_ERROR',
            request.error ?? undefined
          )
        )
      }

      request.onblocked = () => {
        console.warn('Database deletion blocked by open connections')
      }
    })
  }

  /**
   * Check if IndexedDB is supported
   */
  static isSupported(): boolean {
    return typeof indexedDB !== 'undefined'
  }

  /**
   * Get estimated storage quota information
   */
  static async getStorageEstimate(): Promise<StorageEstimate | null> {
    if ('storage' in navigator && 'estimate' in navigator.storage) {
      return navigator.storage.estimate()
    }
    return null
  }
}

/**
 * Singleton instance of the IndexedDB client
 */
let clientInstance: IndexedDBClient | null = null

/**
 * Get the singleton IndexedDB client instance
 */
export function getIndexedDBClient(): IndexedDBClient {
  if (!clientInstance) {
    if (!IndexedDBClient.isSupported()) {
      throw new StorageError(
        'IndexedDB is not supported in this browser',
        'NOT_SUPPORTED'
      )
    }
    clientInstance = new IndexedDBClient()
  }
  return clientInstance
}

/**
 * Reset the singleton instance (useful for testing)
 */
export function resetIndexedDBClient(): void {
  if (clientInstance) {
    clientInstance.close()
    clientInstance = null
  }
}
