declare module 'qrcode.react' {
  import * as React from 'react'
  export const QRCodeSVG: React.FC<{ value?: string; size?: number }>
}

declare module 'canvas-confetti' {
  export default function confetti(opts?: Record<string, unknown>): void
}

declare module 'next/navigation' {
  export function useRouter(): { push: (path: string) => void }
}
