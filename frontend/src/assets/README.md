# HowlerOps Icons

This directory contains the HowlerOps brand icons and logo components.

## Files

- `howlerops-logo.svg` - Full logo version (256x256 viewBox) for large displays
- `howlerops-icon.svg` - Compact icon version (32x32 viewBox) for headers and small spaces
- `HowlerOpsIcon.tsx` - React component for easy integration

## Usage

### React Component

```tsx
import { HowlerOpsIcon } from '@/components/ui/HowlerOpsIcon';

// Small icon for headers
<HowlerOpsIcon size={24} variant="icon" />

// Large logo for splash screens
<HowlerOpsIcon size={128} variant="logo" />

// With custom styling
<HowlerOpsIcon 
  size={32} 
  variant="icon" 
  className="hover:opacity-80 transition-opacity" 
/>
```

### Props

- `size?: number` - Icon size in pixels (default: 32)
- `className?: string` - Additional CSS classes
- `variant?: 'logo' | 'icon'` - Logo (detailed) or icon (simplified) version

### Direct SVG Usage

For non-React contexts, you can use the SVG files directly:

```html
<img src="/src/assets/howlerops-icon.svg" width="24" height="24" alt="HowlerOps" />
```

## Design

The icons feature:
- **Bronze gradient** (#c08b3e to #d7a75e) for the wolf and frame
- **Steel blue gradient** (#6ea2c9 to #9bc0db) for circuit elements  
- **Dark background** (#121a21) for contrast
- **Rounded square frame** with circuit board traces
- **Stylized wolf head** in profile view

The design combines organic (wolf) and technological (circuitry) elements, reflecting the HowlerOps brand identity.
