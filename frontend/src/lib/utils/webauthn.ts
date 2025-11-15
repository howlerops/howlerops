/**
 * WebAuthn Helper Utilities
 *
 * Provides utilities for converting between JSON and WebAuthn API types,
 * handling base64 encoding/decoding for ArrayBuffers used in biometric authentication.
 */

/**
 * Convert base64url string to ArrayBuffer
 */
function base64urlToBuffer(base64url: string): ArrayBuffer {
  // Add padding if needed
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  const padLength = (4 - (base64.length % 4)) % 4
  const padded = base64 + '='.repeat(padLength)

  // Decode base64
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

/**
 * Convert ArrayBuffer to base64url string
 */
function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  const base64 = btoa(binary)
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

/**
 * Parse JSON options from backend into PublicKeyCredentialRequestOptions
 * for navigator.credentials.get()
 */
export function parsePublicKeyRequestOptions(optionsJSON: string): PublicKeyCredentialRequestOptions {
  const options = JSON.parse(optionsJSON)

  return {
    challenge: base64urlToBuffer(options.challenge),
    timeout: options.timeout,
    rpId: options.rpId,
    allowCredentials: options.allowCredentials?.map((cred: any) => ({
      type: cred.type,
      id: base64urlToBuffer(cred.id),
      transports: cred.transports,
    })),
    userVerification: options.userVerification,
  }
}

/**
 * Parse JSON options from backend into PublicKeyCredentialCreationOptions
 * for navigator.credentials.create()
 */
export function parsePublicKeyCreationOptions(optionsJSON: string): PublicKeyCredentialCreationOptions {
  const options = JSON.parse(optionsJSON)

  return {
    challenge: base64urlToBuffer(options.challenge),
    rp: options.rp,
    user: {
      id: base64urlToBuffer(options.user.id),
      name: options.user.name,
      displayName: options.user.displayName,
    },
    pubKeyCredParams: options.pubKeyCredParams,
    timeout: options.timeout,
    authenticatorSelection: options.authenticatorSelection,
    attestation: options.attestation,
    excludeCredentials: options.excludeCredentials?.map((cred: any) => ({
      type: cred.type,
      id: base64urlToBuffer(cred.id),
      transports: cred.transports,
    })),
  }
}

/**
 * Serialize PublicKeyCredential from navigator.credentials.get() to JSON
 * for sending to backend
 */
export function serializeCredentialAssertion(credential: PublicKeyCredential): string {
  const response = credential.response as AuthenticatorAssertionResponse

  return JSON.stringify({
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      authenticatorData: bufferToBase64url(response.authenticatorData),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
    },
  })
}

/**
 * Serialize PublicKeyCredential from navigator.credentials.create() to JSON
 * for sending to backend
 */
export function serializeCredentialAttestation(credential: PublicKeyCredential): string {
  const response = credential.response as AuthenticatorAttestationResponse

  return JSON.stringify({
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToBase64url(response.attestationObject),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
    },
  })
}
