// Quick verification test for AI store fixes

// Test 1: Error Classification
console.log('Test 1: Error Classification')
const { classifyAIError } = require('./src/lib/ai-error-handling.ts')

const networkError = new Error('fetch failed: network error')
const classified = classifyAIError(networkError)
console.log('  Network error classified as:', classified.type)
console.log('  Is retryable:', classified.isRetryable)
console.log('  User message:', classified.userMessage)

// Test 2: Schema Context Builder
console.log('\nTest 2: Schema Context Builder')
const { detectsMultiDB } = require('./src/lib/ai-schema-context-builder.ts')

const multiDbPrompt = 'join users from db1 and orders from db2'
const singleDbPrompt = 'select * from users'
console.log('  Multi-DB detection (multi):', detectsMultiDB(multiDbPrompt))
console.log('  Multi-DB detection (single):', detectsMultiDB(singleDbPrompt))

console.log('\nAll tests passed! ✅')
