# Desktop + Web Shared Codebase Architecture Analysis

## Executive Summary

This document analyzes architectural patterns for applications that need **both a desktop app AND a web dashboard** from a shared codebase. The goal is to identify approaches that allow deploying the same frontend to either a native desktop wrapper (like Wails) or a traditional web server.

---

## Architecture 1: Shared Frontend with Runtime Abstraction Layer

### Concept
Create an abstraction layer between your React frontend and the backend communication mechanism. The frontend calls a unified API interface that routes to either:
- **Desktop mode**: Wails Go bindings (direct function calls)
- **Web mode**: REST/GraphQL API calls over HTTP

### Implementation Pattern

```typescript
// api/abstraction.ts
interface ApiClient {
  getUser(id: string): Promise<User>;
  createProject(data: ProjectInput): Promise<Project>;
  // ... other methods
}

// Desktop implementation (Wails bindings)
class WailsApiClient implements ApiClient {
  async getUser(id: string): Promise<User> {
    return await window.go.main.App.GetUser(id);
  }
}

// Web implementation (REST API)
class RestApiClient implements ApiClient {
  async getUser(id: string): Promise<User> {
    const res = await fetch(`/api/users/${id}`);
    return res.json();
  }
}

// Runtime detection
export const api: ApiClient =
  typeof window.go !== 'undefined'
    ? new WailsApiClient()
    : new RestApiClient();
```

### Pros
- **Maximum code reuse**: 95%+ of frontend code shared
- **Familiar patterns**: Similar to repository/gateway patterns in enterprise apps
- **Incremental migration**: Can migrate from current Wails setup gradually
- **Type safety**: Both implementations share the same interface

### Cons
- **Requires dual backend**: Must maintain Go backend (Wails) AND separate web API server
- **Complexity**: Two codepaths to test and maintain
- **Feature parity challenges**: Some desktop features may not translate to web

### Migration Effort from Current Wails Setup
**Medium** (2-4 weeks)
1. Extract current Wails bindings into interface
2. Create REST API server that mirrors the Go methods
3. Add runtime detection to frontend
4. Test both paths

### Real-World Examples
- [Martin Fowler's Modularizing React Apps](https://martinfowler.com/articles/modularizing-react-apps.html) - API layer abstraction patterns
- [Wails Application Development Guide](https://wails.io/docs/guides/application-development/) - Shows binding patterns that can be abstracted

---

## Architecture 2: Tauri with Web Deployment Strategy

### Concept
Tauri has a better web deployment story than Wails because:
1. Tauri treats the frontend as pure static assets
2. The same frontend can be deployed to any static hosting
3. Tauri provides `invoke()` API that can be conditionally used

### Key Difference from Wails
Wails generates TypeScript bindings that are tightly coupled to the Go runtime. Tauri's approach is more "web-native" - the frontend is truly just a web app that optionally calls native functions.

### Implementation Pattern

```typescript
// Tauri-aware API layer
import { invoke } from '@tauri-apps/api/tauri';

const isTauri = '__TAURI__' in window;

async function getUser(id: string): Promise<User> {
  if (isTauri) {
    return await invoke('get_user', { id });
  }
  return fetch(`/api/users/${id}`).then(r => r.json());
}
```

### Pros
- **Mobile support**: Tauri 2.0 supports iOS and Android
- **Smaller binaries**: ~1MB vs Wails ~8MB
- **Security focused**: Built-in security features
- **Static hosting**: Frontend deploys to any CDN/static host

### Cons
- **Rust backend**: Requires learning Rust for backend logic
- **Longer build times**: 343,135ms vs Wails 12,290ms average
- **Migration effort**: Complete rewrite of Go backend to Rust

### Migration Effort from Current Wails Setup
**High** (1-3 months)
1. Rewrite all Go backend logic in Rust
2. Adapt frontend to use Tauri's invoke API
3. Set up separate web API server (could be Go, Node, etc.)

### Real-World Examples
- [Tauri v2 + Next.js Monorepo Guide](https://melvinoostendorp.nl/blog/tauri-v2-nextjs-monorepo-guide)
- [GitHub Tauri Discussion #3655](https://github.com/tauri-apps/tauri/issues/3655) - Running Tauri apps in browser

---

## Architecture 3: Capacitor Hybrid Approach

### Concept
Use Capacitor as a universal runtime that wraps the same web app for:
- **Web**: Deploy as standard web application
- **Desktop**: Use Electron or Tauri as the wrapper
- **Mobile**: Native iOS/Android containers

### Implementation Pattern

```typescript
// Capacitor plugin detection
import { Capacitor } from '@capacitor/core';
import { Filesystem } from '@capacitor/filesystem';

async function saveFile(data: string, path: string) {
  if (Capacitor.isNativePlatform()) {
    await Filesystem.writeFile({
      path,
      data,
      directory: Directory.Documents,
    });
  } else {
    // Web fallback - download or use IndexedDB
    downloadAsFile(data, path);
  }
}
```

### Pros
- **True "write once"**: Same code runs everywhere
- **Mature ecosystem**: Large plugin library
- **Progressive enhancement**: Web works without native features
- **React-native friendly**: Works with existing React knowledge

### Cons
- **No native desktop support**: Requires additional Electron/Tauri wrapper
- **Performance**: WebView-based, not truly native
- **Plugin quality varies**: Some plugins are better maintained than others

### Migration Effort from Current Wails Setup
**Medium-High** (1-2 months)
1. Add Capacitor to existing React frontend
2. Replace Wails-specific code with Capacitor plugins
3. Create web API server for non-desktop deployment
4. Optionally keep Wails for desktop OR switch to Electron/Tauri

### Real-World Examples
- [Capacitor + React Documentation](https://capacitorjs.com/solution/react)
- [Next.js + Capacitor Tutorial](https://capgo.app/blog/building-a-native-mobile-app-with-nextjs-and-capacitor/)

---

## Architecture 4: Monorepo with Shared UI Components

### Concept
Structure the project as a monorepo with separate apps for web and desktop that share UI components and business logic packages.

### Recommended Structure (Turborepo)

```
packages/
  ui/                 # Shared React components (buttons, forms, etc.)
  api-client/         # Shared API abstraction layer
  types/              # Shared TypeScript types
  business-logic/     # Domain logic, state management
apps/
  web/                # Next.js web application
  desktop/            # Wails/Tauri desktop application
  api/                # Backend API server (if needed)
```

### Implementation Pattern

```typescript
// packages/ui/Button.tsx
export function Button({ onClick, children }: ButtonProps) {
  return <button onClick={onClick}>{children}</button>;
}

// apps/web/pages/dashboard.tsx
import { Button } from '@myapp/ui';
import { api } from '@myapp/api-client';

// apps/desktop/src/App.tsx
import { Button } from '@myapp/ui';
import { api } from '@myapp/api-client'; // Same abstraction!
```

### Pros
- **Clear separation**: Each app has its own entry point
- **Independent deployment**: Web and desktop deploy separately
- **Shared maintenance**: UI fixes apply to both apps
- **Scalable**: Easy to add mobile app later

### Cons
- **Setup complexity**: Monorepo tooling has learning curve
- **Build configuration**: Each app needs its own build setup
- **Some duplication**: App-specific code still duplicated

### Migration Effort from Current Wails Setup
**Medium** (2-4 weeks)
1. Set up Turborepo/Nx monorepo structure
2. Extract shared components into packages
3. Create web app that imports shared packages
4. Keep desktop app (Wails) importing same packages

### Real-World Examples
- [Turborepo + React + Next.js Template](https://medium.com/@beenakumawat002/turborepo-monorepo-in-2025-next-js-react-native-shared-ui-type-safe-api-%EF%B8%8F-6194c83adff9)
- [shadcn/ui Monorepo Setup](https://ui.shadcn.com/docs/monorepo)
- [Universal React Monorepo](https://github.com/gurselcakar/universal-react-monorepo)

---

## Architecture 5: PWA + Desktop Wrapper Hybrid

### Concept
Build the application as a Progressive Web App (PWA) first. The PWA serves as the web version, while the desktop version wraps the same PWA (or its offline-capable version) in Wails/Tauri.

### Implementation Pattern

```typescript
// Service worker for offline capability
// This makes the app work as PWA on web
// And provides offline support in desktop wrapper

// manifest.json for PWA
{
  "name": "HowlerOps",
  "short_name": "Howler",
  "display": "standalone",
  "start_url": "/",
  "theme_color": "#1a1a2e"
}

// Desktop wrapper loads the same app
// Wails config pointing to built PWA assets
{
  "info": {
    "productName": "HowlerOps"
  },
  "frontend": {
    "dir": "./frontend/dist"  // Same build as PWA
  }
}
```

### Pros
- **Single build artifact**: One frontend build for both platforms
- **Offline-first**: PWA patterns benefit desktop too
- **Easy web deployment**: Standard web hosting
- **Installable on web**: Users can "install" PWA without app store

### Cons
- **Limited native access**: PWAs have restricted system access
- **Browser dependency**: Web version limited to browser capabilities
- **Inconsistent PWA support**: Firefox only added Windows PWA in 2025

### Migration Effort from Current Wails Setup
**Low-Medium** (1-2 weeks)
1. Add PWA manifest and service worker to frontend
2. Ensure frontend works standalone (no Wails dependency for UI)
3. Deploy frontend to web hosting
4. Keep Wails wrapper for desktop with enhanced native features

### Real-World Examples
- [PWA + Tauri Discussion](https://www.thinktecture.com/en/contributions/pwa-iwa-tauri-the-future-of-web-based-app-deployment/)
- [Aurelia Desktop + PWA Guide](https://docs.aurelia.io/advanced-scenarios/building-desktop-apps-and-pwa)

---

## Comparison Matrix

| Architecture | Code Reuse | Migration Effort | Complexity | Web Performance | Native Features |
|-------------|------------|------------------|------------|-----------------|-----------------|
| Runtime Abstraction | 95% | Medium | Medium | Good | Full |
| Switch to Tauri | 80% | High | Medium | Good | Full |
| Capacitor Hybrid | 90% | Medium-High | High | Good | Medium |
| Monorepo | 70-80% | Medium | Medium | Excellent | Full |
| PWA + Wrapper | 95% | Low-Medium | Low | Excellent | Limited |

---

## Recommendations

### For HowlerOps Specifically

Given you're currently using Wails with React, I recommend a **phased approach**:

#### Phase 1: Runtime Abstraction Layer (Immediate)
1. Create an API abstraction interface
2. Implement Wails bindings behind this interface
3. This doesn't change the app but prepares for web deployment

#### Phase 2: Monorepo Structure (Short-term)
1. Restructure as Turborepo monorepo
2. Extract shared UI components
3. Create separate `apps/web` with REST API backend

#### Phase 3: PWA Enhancement (Optional)
1. Add PWA capabilities to the shared frontend
2. Desktop gets offline support
3. Web users can "install" the app

### Why Not Tauri?
While Tauri has advantages, the migration cost from Go to Rust is high. The abstraction layer approach lets you keep your Go backend investment while enabling web deployment.

### Why Not Capacitor?
Capacitor is excellent for mobile-first apps but adds unnecessary complexity for desktop-first applications. It would require adding another layer (Electron/Tauri) for proper desktop support anyway.

---

## Implementation Checklist

### Quick Win: Add Runtime Detection

```typescript
// Add to your current Wails React app immediately
export const isDesktopApp = typeof window.go !== 'undefined';

// Use for conditional rendering
{isDesktopApp && <NativeOnlyFeature />}
```

### Next Step: API Abstraction

```typescript
// Create interface matching your current Wails methods
export interface AppApi {
  // Mirror your Go struct methods here
  GetSystemInfo(): Promise<SystemInfo>;
  OpenFile(path: string): Promise<FileContents>;
  // etc.
}
```

This incremental approach lets you validate the architecture before committing to a full restructure.

---

## Sources

### Architecture Patterns
- [React Architecture Patterns 2025](https://www.geeksforgeeks.org/reactjs/react-architecture-pattern-and-best-practices/)
- [Modularizing React Apps - Martin Fowler](https://martinfowler.com/articles/modularizing-react-apps.html)
- [Feature-Based Architecture](https://www.bacancytechnology.com/blog/react-architecture-patterns-and-best-practices)

### Framework Comparisons
- [Tauri vs Wails Discussion](https://github.com/tauri-apps/tauri/discussions/3521)
- [Web-to-Desktop Framework Comparison](https://github.com/Elanis/web-to-desktop-framework-comparison)
- [Electron Alternatives](https://brainhub.eu/library/electron-alternatives-javascript-frameworks-for-desktop-apps)

### Monorepo Patterns
- [Turborepo Monorepo 2025](https://medium.com/@beenakumawat002/turborepo-monorepo-in-2025-next-js-react-native-shared-ui-type-safe-api-%EF%B8%8F-6194c83adff9)
- [Nx React Monorepo Guide](https://medium.com/@sakshijaiswal0310/building-a-scalable-react-monorepo-with-nx-and-shadcn-ui-a-complete-implementation-guide-96c2bb1b42e8)
- [shadcn/ui Monorepo](https://ui.shadcn.com/docs/monorepo)

### PWA & Hybrid Approaches
- [PWA Future 2025](https://brainhub.eu/library/is-pwa-the-future)
- [PWA + Tauri Discussion](https://www.thinktecture.com/en/contributions/pwa-iwa-tauri-the-future-of-web-based-app-deployment/)
- [Capacitor + React](https://capacitorjs.com/solution/react)

### Wails & Tauri Documentation
- [Wails Application Development](https://wails.io/docs/guides/application-development/)
- [Tauri Frontend Configuration](https://v2.tauri.app/start/frontend/)
- [Tauri v2 + Next.js Monorepo](https://melvinoostendorp.nl/blog/tauri-v2-nextjs-monorepo-guide)
