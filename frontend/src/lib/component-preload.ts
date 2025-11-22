/**
 * Component Preloading Utilities
 *
 * Provides utilities for preloading lazy-loaded React components to improve perceived performance.
 * Components can be preloaded on user interaction (hover, focus) before they're actually needed.
 */

type PreloadableComponent = () => Promise<{ default: React.ComponentType<unknown> }>

const preloadedComponents = new Set<PreloadableComponent>()

/**
 * Preload a lazy-loaded component
 *
 * @param componentLoader - The lazy component loader function
 * @returns Promise that resolves when the component is loaded
 *
 * @example
 * ```tsx
 * const MyComponent = lazy(() => import('./MyComponent'))
 *
 * // Preload on hover
 * <button onMouseEnter={() => preloadComponent(MyComponent)}>
 *   Open Feature
 * </button>
 * ```
 */
export async function preloadComponent(componentLoader: PreloadableComponent): Promise<void> {
  // Only preload once
  if (preloadedComponents.has(componentLoader)) {
    return
  }

  try {
    preloadedComponents.add(componentLoader)
    await componentLoader()
  } catch (error) {
    // Remove from cache on error so it can be retried
    preloadedComponents.delete(componentLoader)
    console.warn('Failed to preload component:', error)
  }
}

/**
 * Create a preload handler for use with event handlers
 *
 * @param componentLoader - The lazy component loader function
 * @returns Event handler function that preloads the component
 *
 * @example
 * ```tsx
 * const MyComponent = lazy(() => import('./MyComponent'))
 *
 * <button onMouseEnter={createPreloadHandler(MyComponent)}>
 *   Open Feature
 * </button>
 * ```
 */
export function createPreloadHandler(componentLoader: PreloadableComponent) {
  return () => {
    void preloadComponent(componentLoader)
  }
}

/**
 * Preload multiple components in parallel
 *
 * @param componentLoaders - Array of lazy component loader functions
 * @returns Promise that resolves when all components are loaded
 *
 * @example
 * ```tsx
 * // Preload related components together
 * preloadComponents([AIQueryTab, SchemaVisualizer, ReportBuilder])
 * ```
 */
export async function preloadComponents(componentLoaders: PreloadableComponent[]): Promise<void> {
  await Promise.all(componentLoaders.map(loader => preloadComponent(loader)))
}

/**
 * Check if a component has been preloaded
 *
 * @param componentLoader - The lazy component loader function
 * @returns true if the component has been preloaded
 */
export function isComponentPreloaded(componentLoader: PreloadableComponent): boolean {
  return preloadedComponents.has(componentLoader)
}

/**
 * Clear the preload cache (useful for testing or memory management)
 */
export function clearPreloadCache(): void {
  preloadedComponents.clear()
}
