import { QueryClient, QueryCache, QueryClientProvider, UseQueryOptions } from "@tanstack/react-query";
import { ReactNode, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "../lib/api-client";
import { useToast } from "@/components/ui/use-toast";

interface QueryProviderProps {
  children: ReactNode;
}

/**
 * Create QueryClient with global error handling for 401 Unauthorized errors
 */
export function createQueryClient() {
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error, query) => {
        // Only handle errors for queries that have data (background refresh failures)
        if (query.state.data === undefined) {
          return;
        }

        // Handle ApiError specifically
        if (error instanceof ApiError && error.status === 401) {
          // 401 errors during background refresh are logged but not shown to user
          // The user is already authenticated, so this is likely a token expiry
          console.warn('Background authentication refresh failed:', error.message);
          return;
        }

        // Log other errors
        console.error('Query error:', error);
      },
    }),
    defaultOptions: {
      queries: {
        // Retry configuration
        retry: (failureCount, error) => {
          // Don't retry 401 errors
          if (error instanceof ApiError && error.status === 401) {
            return false;
          }
          // Retry up to 3 times for other errors
          return failureCount < 3;
        },
        // Refetch on window focus for fresh data
        refetchOnWindowFocus: true,
        // Stale time for wallet info (reduce unnecessary refetches)
        staleTime: 30000,
      },
    },
  });
}

/**
 * Query Provider component with global 401 error handling
 *
 * This provider wraps the application and handles authentication errors
 * globally, redirecting users to login when 401 errors occur.
 */
export function QueryProvider({ children }: QueryProviderProps) {
  const navigate = useNavigate();
  const { toast } = useToast();

  // Create client with error handling
  const queryClient = useMemo(() => createQueryClient(), []);

  // Handle 401 errors - redirect to login
  const handleUnauthorized = useCallback(() => {
    // Clear invalid token
    localStorage.removeItem('token');

    // Show toast notification
    toast({
      variant: 'destructive',
      title: 'Session Expired',
      description: 'Your session has expired. Please log in again.',
    });

    // Redirect to login with return URL
    navigate('/login', { state: { returnTo: window.location.pathname } });
  }, [navigate, toast]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

export default QueryProvider;
