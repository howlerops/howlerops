/**
 * Preference Repository
 *
 * Manages UI preferences and settings with:
 * - User-specific preferences
 * - Device-specific settings
 * - Category-based organization
 * - Type-safe get/set operations
 *
 * @module lib/storage/repositories/preference-repository
 */

import { getIndexedDBClient } from '../indexeddb-client'
import {
  STORE_NAMES,
  type UIPreferenceRecord,
  type CreateInput,
  NotFoundError,
} from '@/types/storage'

/**
 * Preference value types
 */
export type PreferenceValue = string | number | boolean | object | null

/**
 * Preference categories
 */
export enum PreferenceCategory {
  EDITOR = 'editor',
  THEME = 'theme',
  LAYOUT = 'layout',
  BEHAVIOR = 'behavior',
  DISPLAY = 'display',
  ACCESSIBILITY = 'accessibility',
  NOTIFICATIONS = 'notifications',
  PRIVACY = 'privacy',
  PERFORMANCE = 'performance',
  CUSTOM = 'custom',
}

/**
 * Repository for managing UI preferences
 */
export class PreferenceRepository {
  private client = getIndexedDBClient()
  private storeName = STORE_NAMES.UI_PREFERENCES
  private deviceId: string

  constructor() {
    // Get or create device ID
    this.deviceId = this.getDeviceId()
  }

  /**
   * Get or create a unique device ID
   */
  private getDeviceId(): string {
    const key = 'sql-studio-device-id'
    let deviceId = localStorage.getItem(key)

    if (!deviceId) {
      deviceId = crypto.randomUUID()
      localStorage.setItem(key, deviceId)
    }

    return deviceId
  }

  /**
   * Create a new preference record
   */
  async create(
    data: CreateInput<UIPreferenceRecord>
  ): Promise<UIPreferenceRecord> {
    const now = new Date()
    const record: UIPreferenceRecord = {
      id: data.id || crypto.randomUUID(),
      user_id: data.user_id,
      key: data.key,
      value: data.value,
      category: data.category,
      device_id: data.device_id,
      updated_at: data.updated_at ?? now,
      synced: data.synced ?? false,
      sync_version: data.sync_version ?? 0,
    }

    await this.client.put(this.storeName, record)
    return record
  }

  /**
   * Get a preference by ID
   */
  async get(id: string): Promise<UIPreferenceRecord | null> {
    return this.client.get<UIPreferenceRecord>(this.storeName, id)
  }

  /**
   * Set a user preference
   */
  async setUserPreference(
    userId: string,
    key: string,
    value: PreferenceValue,
    category: string = PreferenceCategory.CUSTOM
  ): Promise<UIPreferenceRecord> {
    // Check if preference already exists
    const existing = await this.getUserPreference(userId, key)

    if (existing) {
      const updated: UIPreferenceRecord = {
        ...existing,
        value,
        category,
        updated_at: new Date(),
        synced: false, // Mark as unsynced
      }
      await this.client.put(this.storeName, updated)
      return updated
    }

    // Create new preference
    return this.create({
      user_id: userId,
      key,
      value,
      category,
    })
  }

  /**
   * Get a user preference by key
   */
  async getUserPreference(
    userId: string,
    key: string
  ): Promise<UIPreferenceRecord | null> {
    const records = await this.client.getAll<UIPreferenceRecord>(
      this.storeName,
      {
        index: 'user_key',
        range: IDBKeyRange.only([userId, key]),
      }
    )

    // Return first match (should only be one)
    return records[0] ?? null
  }

  /**
   * Get user preference value (typed)
   */
  async getUserPreferenceValue<T = PreferenceValue>(
    userId: string,
    key: string,
    defaultValue?: T
  ): Promise<T | undefined> {
    const pref = await this.getUserPreference(userId, key)
    if (pref) {
      return pref.value as T
    }
    return defaultValue
  }

  /**
   * Set a device-specific preference
   */
  async setDevicePreference(
    key: string,
    value: PreferenceValue,
    category: string = PreferenceCategory.CUSTOM
  ): Promise<UIPreferenceRecord> {
    // Check if preference already exists
    const existing = await this.getDevicePreference(key)

    if (existing) {
      const updated: UIPreferenceRecord = {
        ...existing,
        value,
        category,
        updated_at: new Date(),
        synced: false, // Device preferences typically don't sync
      }
      await this.client.put(this.storeName, updated)
      return updated
    }

    // Create new preference
    return this.create({
      device_id: this.deviceId,
      key,
      value,
      category,
      synced: false, // Device preferences don't sync
    })
  }

  /**
   * Get a device-specific preference
   */
  async getDevicePreference(key: string): Promise<UIPreferenceRecord | null> {
    const records = await this.client.getAll<UIPreferenceRecord>(
      this.storeName,
      {
        index: 'device_key',
        range: IDBKeyRange.only([this.deviceId, key]),
      }
    )

    return records[0] ?? null
  }

  /**
   * Get device preference value (typed)
   */
  async getDevicePreferenceValue<T = PreferenceValue>(
    key: string,
    defaultValue?: T
  ): Promise<T | undefined> {
    const pref = await this.getDevicePreference(key)
    if (pref) {
      return pref.value as T
    }
    return defaultValue
  }

  /**
   * Get all preferences for a user
   */
  async getAllUserPreferences(
    userId: string
  ): Promise<UIPreferenceRecord[]> {
    return this.client.getAll<UIPreferenceRecord>(this.storeName, {
      index: 'user_id',
      range: IDBKeyRange.only(userId),
    })
  }

  /**
   * Get all device preferences
   */
  async getAllDevicePreferences(): Promise<UIPreferenceRecord[]> {
    return this.client.getAll<UIPreferenceRecord>(this.storeName, {
      index: 'device_id',
      range: IDBKeyRange.only(this.deviceId),
    })
  }

  /**
   * Get preferences by category
   */
  async getByCategory(
    category: string,
    userId?: string
  ): Promise<UIPreferenceRecord[]> {
    const allPrefs = await this.client.getAll<UIPreferenceRecord>(
      this.storeName,
      {
        index: 'category',
        range: IDBKeyRange.only(category),
      }
    )

    if (userId) {
      return allPrefs.filter((p) => p.user_id === userId)
    }

    return allPrefs
  }

  /**
   * Delete a preference
   */
  async delete(id: string): Promise<void> {
    await this.client.delete(this.storeName, id)
  }

  /**
   * Delete user preference by key
   */
  async deleteUserPreference(userId: string, key: string): Promise<void> {
    const pref = await this.getUserPreference(userId, key)
    if (pref) {
      await this.delete(pref.id)
    }
  }

  /**
   * Delete device preference by key
   */
  async deleteDevicePreference(key: string): Promise<void> {
    const pref = await this.getDevicePreference(key)
    if (pref) {
      await this.delete(pref.id)
    }
  }

  /**
   * Clear all preferences for a user
   */
  async clearUserPreferences(userId: string): Promise<number> {
    const prefs = await this.getAllUserPreferences(userId)
    await Promise.all(prefs.map((p) => this.delete(p.id)))
    return prefs.length
  }

  /**
   * Clear all device preferences
   */
  async clearDevicePreferences(): Promise<number> {
    const prefs = await this.getAllDevicePreferences()
    await Promise.all(prefs.map((p) => this.delete(p.id)))
    return prefs.length
  }

  /**
   * Get unsynced preferences for server sync
   */
  async getUnsynced(limit = 100): Promise<UIPreferenceRecord[]> {
    return this.client.getAll<UIPreferenceRecord>(this.storeName, {
      index: 'synced',
      range: IDBKeyRange.only(false),
      limit,
    })
  }

  /**
   * Mark preferences as synced
   */
  async markSynced(ids: string[], syncVersion: number): Promise<void> {
    const records = await Promise.all(
      ids.map((id) => this.client.get<UIPreferenceRecord>(this.storeName, id))
    )

    await Promise.all(
      records.map((record) => {
        if (record) {
          return this.client.put(this.storeName, {
            ...record,
            synced: true,
            sync_version: syncVersion,
          })
        }
      })
    )
  }

  /**
   * Bulk set preferences (useful for initial setup)
   */
  async bulkSet(
    preferences: Array<{
      key: string
      value: PreferenceValue
      category?: string
      userId?: string
    }>
  ): Promise<UIPreferenceRecord[]> {
    const records = await Promise.all(
      preferences.map((pref) => {
        if (pref.userId) {
          return this.setUserPreference(
            pref.userId,
            pref.key,
            pref.value,
            pref.category
          )
        } else {
          return this.setDevicePreference(
            pref.key,
            pref.value,
            pref.category
          )
        }
      })
    )

    return records
  }

  /**
   * Get preferences as key-value map
   */
  async getPreferencesMap(
    userId?: string,
    deviceOnly = false
  ): Promise<Record<string, PreferenceValue>> {
    const prefs = deviceOnly
      ? await this.getAllDevicePreferences()
      : userId
      ? await this.getAllUserPreferences(userId)
      : await this.client.getAll<UIPreferenceRecord>(this.storeName)

    const map: Record<string, PreferenceValue> = {}
    prefs.forEach((pref) => {
      map[pref.key] = pref.value
    })

    return map
  }

  /**
   * Export preferences for backup
   */
  async exportPreferences(
    userId?: string
  ): Promise<Array<{ key: string; value: PreferenceValue; category: string }>> {
    const prefs = userId
      ? await this.getAllUserPreferences(userId)
      : await this.getAllDevicePreferences()

    return prefs.map((pref) => ({
      key: pref.key,
      value: pref.value,
      category: pref.category,
    }))
  }

  /**
   * Import preferences from backup
   */
  async importPreferences(
    preferences: Array<{
      key: string
      value: PreferenceValue
      category: string
    }>,
    userId?: string
  ): Promise<number> {
    const records = await this.bulkSet(
      preferences.map((pref) => ({
        ...pref,
        userId,
      }))
    )

    return records.length
  }

  /**
   * Get total count of preferences
   */
  async count(options?: { userId?: string }): Promise<number> {
    if (options?.userId) {
      return this.client.count(this.storeName, {
        index: 'user_id',
        range: IDBKeyRange.only(options.userId),
      })
    }

    return this.client.count(this.storeName)
  }
}

/**
 * Singleton instance
 */
let repositoryInstance: PreferenceRepository | null = null

/**
 * Get singleton instance of PreferenceRepository
 */
export function getPreferenceRepository(): PreferenceRepository {
  if (!repositoryInstance) {
    repositoryInstance = new PreferenceRepository()
  }
  return repositoryInstance
}
