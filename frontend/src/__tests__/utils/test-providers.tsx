import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider } from '@/components/theme-provider';

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

// Test wrapper component
interface AllTheProvidersProps {
  children: React.ReactNode;
  queryClient?: QueryClient;
  initialRoute?: string;
  store?: MockStore;
}

export const AllTheProviders: React.FC<AllTheProvidersProps> = ({
  children,
  queryClient,
  initialRoute = '/',
}) => {
  const testQueryClient = queryClient || new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: Infinity,
      },
      mutations: {
        retry: false,
      },
    },
  });

  // Mock the router with initial route
  window.history.pushState({}, 'Test page', initialRoute);

  return (
    <QueryClientProvider client={testQueryClient}>
      <BrowserRouter>
        <ThemeProvider defaultTheme="light" storageKey="test-theme">
          {children}
        </ThemeProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
};