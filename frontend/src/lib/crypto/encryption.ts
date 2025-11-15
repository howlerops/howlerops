/**
 * Frontend encryption utilities using Web Crypto API
 * Implements AES-256-GCM encryption with PBKDF2 key derivation
 *
 * Security properties:
 * - Zero-knowledge: Server never sees plaintext passwords
 * - PBKDF2-SHA256 with 600,000 iterations (OWASP 2023)
 * - AES-256-GCM authenticated encryption
 * - Unique salt per user, unique IV per encryption
 */

const PBKDF2_ITERATIONS = 600_000; // OWASP 2023 recommendation
const SALT_LENGTH = 32; // 256 bits
const IV_LENGTH = 12; // 96 bits for GCM

export interface EncryptedData {
  ciphertext: string; // Base64
  iv: string; // Base64
  authTag: string; // Base64
}

export interface EncryptedMasterKey extends EncryptedData {
  salt: string; // Base64
  iterations: number;
}

/**
 * Generate a random salt for PBKDF2
 */
export function generateSalt(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
}

/**
 * Generate a random IV for AES-GCM
 */
export function generateIV(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(IV_LENGTH));
}

/**
 * Convert ArrayBuffer or Uint8Array to Base64 string
 */
function arrayBufferToBase64(buffer: ArrayBuffer | Uint8Array): string {
  const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

/**
 * Convert Base64 string to ArrayBuffer
 */
function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Derive a cryptographic key from a user password using PBKDF2
 *
 * @param password - User's login password
 * @param salt - Random salt (unique per user)
 * @returns CryptoKey suitable for AES-GCM encryption
 */
export async function deriveKeyFromPassword(
  password: string,
  salt: Uint8Array
): Promise<CryptoKey> {
  // Import password as key material
  const passwordKey = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );

  // Derive key using PBKDF2
  return crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt.buffer as ArrayBuffer,
      iterations: PBKDF2_ITERATIONS,
      hash: 'SHA-256',
    },
    passwordKey,
    { name: 'AES-GCM', length: 256 },
    false, // Not extractable
    ['encrypt', 'decrypt']
  );
}

/**
 * Generate a random master key for encrypting database passwords
 *
 * @returns CryptoKey (AES-256)
 */
export async function generateMasterKey(): Promise<CryptoKey> {
  return crypto.subtle.generateKey(
    { name: 'AES-GCM', length: 256 },
    true, // Extractable (so we can encrypt it)
    ['encrypt', 'decrypt']
  );
}

/**
 * Encrypt the master key using the user's password-derived key
 *
 * @param masterKey - The master key to encrypt
 * @param userPassword - User's login password
 * @returns Encrypted master key with metadata
 */
export async function encryptMasterKey(
  masterKey: CryptoKey,
  userPassword: string
): Promise<EncryptedMasterKey> {
  // Generate salt for this user
  const salt = generateSalt();

  // Derive key from user's password
  const userKey = await deriveKeyFromPassword(userPassword, salt);

  // Export master key as raw bytes
  const masterKeyBytes = await crypto.subtle.exportKey('raw', masterKey);

  // Generate IV
  const iv = generateIV();

  // Encrypt master key with user's derived key
  const encryptedData = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: iv.buffer as ArrayBuffer },
    userKey,
    masterKeyBytes
  );

  // AES-GCM produces ciphertext with auth tag appended
  // We need to split them for storage
  const encryptedBytes = new Uint8Array(encryptedData);
  const ciphertext = encryptedBytes.slice(0, -16); // Everything except last 16 bytes
  const authTag = encryptedBytes.slice(-16); // Last 16 bytes

  return {
    ciphertext: arrayBufferToBase64(ciphertext),
    iv: arrayBufferToBase64(iv),
    authTag: arrayBufferToBase64(authTag),
    salt: arrayBufferToBase64(salt),
    iterations: PBKDF2_ITERATIONS,
  };
}

/**
 * Decrypt the master key using the user's password
 *
 * @param encryptedKey - Encrypted master key data
 * @param userPassword - User's login password
 * @returns Decrypted master key
 */
export async function decryptMasterKey(
  encryptedKey: EncryptedMasterKey,
  userPassword: string
): Promise<CryptoKey> {
  // Derive key from user's password
  const salt = new Uint8Array(base64ToArrayBuffer(encryptedKey.salt));
  const userKey = await deriveKeyFromPassword(userPassword, salt);

  // Reconstruct encrypted data (ciphertext + auth tag)
  const ciphertext = new Uint8Array(base64ToArrayBuffer(encryptedKey.ciphertext));
  const authTag = new Uint8Array(base64ToArrayBuffer(encryptedKey.authTag));
  const encryptedData = new Uint8Array(ciphertext.length + authTag.length);
  encryptedData.set(ciphertext);
  encryptedData.set(authTag, ciphertext.length);

  // Decrypt master key
  const iv = new Uint8Array(base64ToArrayBuffer(encryptedKey.iv));
  const masterKeyBytes = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv },
    userKey,
    encryptedData
  );

  // Import as CryptoKey
  return crypto.subtle.importKey(
    'raw',
    masterKeyBytes,
    { name: 'AES-GCM', length: 256 },
    false, // Not extractable
    ['encrypt', 'decrypt']
  );
}

/**
 * Encrypt a database password using the master key
 *
 * @param password - Database password (plaintext)
 * @param masterKey - User's master key
 * @returns Encrypted password with IV and auth tag
 */
export async function encryptPassword(
  password: string,
  masterKey: CryptoKey
): Promise<EncryptedData> {
  const iv = generateIV();
  const passwordBytes = new TextEncoder().encode(password);

  const encryptedData = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: iv.buffer as ArrayBuffer },
    masterKey,
    passwordBytes
  );

  // Split ciphertext and auth tag
  const encryptedBytes = new Uint8Array(encryptedData);
  const ciphertext = encryptedBytes.slice(0, -16);
  const authTag = encryptedBytes.slice(-16);

  return {
    ciphertext: arrayBufferToBase64(ciphertext),
    iv: arrayBufferToBase64(iv),
    authTag: arrayBufferToBase64(authTag),
  };
}

/**
 * Decrypt a database password using the master key
 *
 * @param encryptedData - Encrypted password data
 * @param masterKey - User's master key
 * @returns Decrypted password (plaintext)
 */
export async function decryptPassword(
  encryptedData: EncryptedData,
  masterKey: CryptoKey
): Promise<string> {
  // Reconstruct encrypted data (ciphertext + auth tag)
  const ciphertext = new Uint8Array(base64ToArrayBuffer(encryptedData.ciphertext));
  const authTag = new Uint8Array(base64ToArrayBuffer(encryptedData.authTag));
  const encrypted = new Uint8Array(ciphertext.length + authTag.length);
  encrypted.set(ciphertext);
  encrypted.set(authTag, ciphertext.length);

  // Decrypt
  const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
  const passwordBytes = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv },
    masterKey,
    encrypted
  );

  return new TextDecoder().decode(passwordBytes);
}

/**
 * Export master key as Base64 string for session storage (use with caution!)
 * Only use for temporary session caching in memory, never persist to disk
 */
export async function exportMasterKeyToBase64(masterKey: CryptoKey): Promise<string> {
  const keyBytes = await crypto.subtle.exportKey('raw', masterKey);
  return arrayBufferToBase64(keyBytes);
}

/**
 * Import master key from Base64 string (for session restoration)
 */
export async function importMasterKeyFromBase64(base64Key: string): Promise<CryptoKey> {
  const keyBytes = base64ToArrayBuffer(base64Key);
  return crypto.subtle.importKey(
    'raw',
    keyBytes,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}
