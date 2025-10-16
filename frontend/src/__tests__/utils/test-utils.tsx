import React from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { faker } from '@faker-js/faker';
import { AllTheProviders } from './test-providers';

// Mock store type (adjust based on your actual store)
interface MockStore {
  user: unknown;
  connections: unknown[];
  currentConnection: string | null;
  queryResults: unknown;
  setUser: (user: unknown) => void;
  setConnections: (connections: unknown[]) => void;
  setCurrentConnection: (id: string | null) => void;
  setQueryResults: (results: unknown) => void;
}

// Create a mock store
export const createMockStore = (initialState?: Partial<MockStore>): MockStore => ({
  user: null,
  connections: [],
  currentConnection: null,
  queryResults: null,
  setUser: vi.fn(),
  setConnections: vi.fn(),
  setCurrentConnection: vi.fn(),
  setQueryResults: vi.fn(),
  ...initialState,
});


// Custom render function
const customRender = (
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'> & {
    queryClient?: QueryClient;
    initialRoute?: string;
    store?: MockStore;
  }
) => {
  const { queryClient, initialRoute, ...renderOptions } = options || {};

  return render(ui, {
    wrapper: ({ children }) => (
      <AllTheProviders
        queryClient={queryClient}
        initialRoute={initialRoute}
      >
        {children}
      </AllTheProviders>
    ),
    ...renderOptions,
  });
};

// Mock data generators
export const mockUser = (overrides?: Partial<Record<string, unknown>>) => ({
  id: faker.string.uuid(),
  username: faker.internet.username(),
  email: faker.internet.email(),
  role: faker.helpers.arrayElement(['admin', 'user', 'readonly']),
  createdAt: faker.date.past(),
  ...overrides,
});

export const mockConnection = (overrides?: Partial<Record<string, unknown>>) => ({
  id: faker.string.uuid(),
  name: faker.company.name(),
  type: faker.helpers.arrayElement(['postgres', 'mysql', 'sqlite']),
  host: faker.internet.ip(),
  port: faker.number.int({ min: 3000, max: 5432 }),
  database: faker.database.mongodbObjectId(),
  username: faker.internet.username(),
  status: faker.helpers.arrayElement(['connected', 'disconnected', 'error']),
  createdAt: faker.date.past(),
  ...overrides,
});

export const mockQueryResult = (overrides?: Partial<Record<string, unknown>>) => ({
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, () => ({
    id: faker.number.int(),
    name: faker.person.fullName(),
    email: faker.internet.email(),
    status: faker.helpers.arrayElement(['active', 'inactive']),
  })),
  columns: [
    { name: 'id', type: 'integer' },
    { name: 'name', type: 'varchar' },
    { name: 'email', type: 'varchar' },
    { name: 'status', type: 'varchar' },
  ],
  executionTime: faker.number.int({ min: 10, max: 1000 }),
  rowCount: faker.number.int({ min: 1, max: 100 }),
  ...overrides,
});

export const mockTableSchema = (overrides?: Partial<Record<string, unknown>>) => ({
  name: faker.database.collation(),
  columns: Array.from({ length: faker.number.int({ min: 3, max: 8 }) }, () => ({
    name: faker.database.column(),
    type: faker.helpers.arrayElement(['INTEGER', 'VARCHAR(255)', 'TEXT', 'TIMESTAMP']),
    nullable: faker.datatype.boolean(),
    primaryKey: faker.datatype.boolean(),
    defaultValue: faker.helpers.maybe(() => faker.string.sample()),
  })),
  indexes: Array.from({ length: faker.number.int({ min: 0, max: 3 }) }, () => ({
    name: `idx_${faker.database.column()}`,
    columns: [faker.database.column()],
    unique: faker.datatype.boolean(),
  })),
  ...overrides,
});

// Event helpers
export const mockEvent = <T extends Event>(
  type: string,
  properties: Partial<T> = {}
): T => {
  const event = new Event(type) as T;
  Object.assign(event, properties);
  return event;
};

export const mockKeyboardEvent = (
  key: string,
  options: Partial<KeyboardEvent> = {}
): KeyboardEvent => {
  return mockEvent('keydown', {
    key,
    code: `Key${key.toUpperCase()}`,
    keyCode: key.charCodeAt(0),
    ...options,
  });
};

export const mockMouseEvent = (
  type: string = 'click',
  options: Partial<MouseEvent> = {}
): MouseEvent => {
  return mockEvent(type, {
    button: 0,
    buttons: 1,
    clientX: 100,
    clientY: 100,
    ...options,
  });
};

// Wait utilities
export const waitForCondition = async (
  condition: () => boolean | Promise<boolean>,
  timeout: number = 5000,
  interval: number = 100
): Promise<void> => {
  const startTime = Date.now();

  while (Date.now() - startTime < timeout) {
    if (await condition()) {
      return;
    }
    await new Promise(resolve => setTimeout(resolve, interval));
  }

  throw new Error(`Condition not met within ${timeout}ms`);
};

export const sleep = (ms: number): Promise<void> => {
  return new Promise(resolve => setTimeout(resolve, ms));
};

// Custom matchers (if needed)
export const expectToBeInDocument = (element: HTMLElement | null) => {
  expect(element).toBeInTheDocument();
};

export const expectToHaveClass = (element: HTMLElement | null, className: string) => {
  expect(element).toHaveClass(className);
};

export const expectToBeVisible = (element: HTMLElement | null) => {
  expect(element).toBeVisible();
};

// Re-export testing utilities (non-components only)
export { render as defaultRender, screen, waitFor, act, fireEvent, cleanup, within, getByRole, getByText, getByLabelText, getByTestId, getByDisplayValue, getByAltText, getByTitle, queryByRole, queryByText, queryByLabelText, queryByTestId, queryByDisplayValue, queryByAltText, queryByTitle, findByRole, findByText, findByLabelText, findByTestId, findByDisplayValue, findByAltText, findByTitle } from '@testing-library/react';
export { default as userEvent } from '@testing-library/user-event';

// Override the default render
export { customRender as render };