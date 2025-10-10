import '@testing-library/jest-dom';
import { afterEach, beforeAll, afterAll } from 'vitest';
import { cleanup } from '@testing-library/react';
import { setupServer } from 'msw/node';
import { HttpResponse, http } from 'msw';

// Mock handlers for API calls
export const handlers = [
  // Auth endpoints
  http.post('/api/auth/login', () => {
    return HttpResponse.json({
      token: 'mock-jwt-token',
      user: {
        id: '1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin',
      },
    });
  }),

  http.post('/api/auth/logout', () => {
    return HttpResponse.json({ message: 'Logged out successfully' });
  }),

  // Connections endpoints
  http.get('/api/connections', () => {
    return HttpResponse.json([
      {
        id: '1',
        name: 'Test Database',
        type: 'postgres',
        host: 'localhost',
        port: 5432,
        database: 'testdb',
        status: 'connected',
      },
    ]);
  }),

  http.post('/api/connections', () => {
    return HttpResponse.json({
      id: '2',
      name: 'New Connection',
      type: 'mysql',
      status: 'connected',
    }, { status: 201 });
  }),

  // Queries endpoints
  http.post('/api/queries/execute', () => {
    return HttpResponse.json({
      rows: [
        { id: 1, name: 'John Doe', email: 'john@example.com' },
        { id: 2, name: 'Jane Smith', email: 'jane@example.com' },
      ],
      columns: [
        { name: 'id', type: 'integer' },
        { name: 'name', type: 'varchar' },
        { name: 'email', type: 'varchar' },
      ],
      executionTime: 150,
      rowCount: 2,
    });
  }),

  // Schema endpoints
  http.get('/api/connections/:id/schema', () => {
    return HttpResponse.json({
      tables: [
        {
          name: 'users',
          columns: [
            { name: 'id', type: 'INTEGER', primaryKey: true },
            { name: 'name', type: 'VARCHAR(255)', nullable: false },
            { name: 'email', type: 'VARCHAR(255)', nullable: false },
          ],
        },
      ],
    });
  }),
];

// Setup MSW server
export const server = setupServer(...handlers);

// Start server before all tests
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
});

// Clean up after each test case
afterEach(() => {
  cleanup();
  server.resetHandlers();
});

// Close server after all tests
afterAll(() => {
  server.close();
});

// Global test utilities
declare global {
  interface JestAssertion<T = unknown> {
    toBeInTheDocument(): T;
    toHaveClass(className: string): T;
    toHaveStyle(style: string | object): T;
  }
}

// Mock IntersectionObserver
Object.defineProperty(window, 'IntersectionObserver', {
  writable: true,
  value: class IntersectionObserver {
    constructor(callback: IntersectionObserverCallback) {
      this.callback = callback;
    }
    callback: IntersectionObserverCallback;
    observe() { return null; }
    disconnect() { return null; }
    unobserve() { return null; }
  },
});

// Mock ResizeObserver
Object.defineProperty(window, 'ResizeObserver', {
  writable: true,
  value: class ResizeObserver {
    constructor(callback: ResizeObserverCallback) {
      this.callback = callback;
    }
    callback: ResizeObserverCallback;
    observe() { return null; }
    disconnect() { return null; }
    unobserve() { return null; }
  },
});

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => {},
  }),
});

// Mock scrollTo
Object.defineProperty(window, 'scrollTo', {
  writable: true,
  value: () => {},
});

// Note: Clipboard API is handled by userEvent.setup() in individual tests