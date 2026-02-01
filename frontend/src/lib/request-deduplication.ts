// Tracks in-flight requests to prevent duplicates
const inFlightRequests = new Map<string, Promise<unknown>>()

export async function dedupedRequest<T>(
  key: string,
  requestFn: () => Promise<T>
): Promise<T> {
  if (inFlightRequests.has(key)) {
    return inFlightRequests.get(key) as Promise<T>
  }

  const promise = requestFn().finally(() => {
    inFlightRequests.delete(key)
  })

  inFlightRequests.set(key, promise)
  return promise
}
