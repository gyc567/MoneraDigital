import { describe, it, expect, beforeEach } from 'vitest';

/**
 * Utility functions for safe localStorage operations
 */

export function safeSetUser(user: unknown): void {
  if (user && typeof user === 'object') {
    localStorage.setItem('user', JSON.stringify(user));
  }
}

export function safeGetUser(): { email: string; id?: number } | null {
  const savedUser = localStorage.getItem('user');
  if (!savedUser || savedUser === 'undefined' || savedUser === 'null') {
    // Clean up invalid values
    localStorage.removeItem('user');
    return null;
  }
  try {
    return JSON.parse(savedUser);
  } catch (e) {
    console.error('Failed to parse user data:', e);
    localStorage.removeItem('user');
    return null;
  }
}

export function clearAuthData(): void {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
}

describe('localStorage utils', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('safeSetUser', () => {
    it('should store valid user object', () => {
      const user = { email: 'test@example.com', id: 1 };
      safeSetUser(user);
      expect(localStorage.getItem('user')).toBe(JSON.stringify(user));
    });

    it('should not store undefined', () => {
      safeSetUser(undefined);
      expect(localStorage.getItem('user')).toBeNull();
    });

    it('should not store null', () => {
      safeSetUser(null);
      expect(localStorage.getItem('user')).toBeNull();
    });

    it('should not store primitive values', () => {
      safeSetUser('string');
      expect(localStorage.getItem('user')).toBeNull();
    });
  });

  describe('safeGetUser', () => {
    it('should return parsed user object', () => {
      const user = { email: 'test@example.com', id: 1 };
      localStorage.setItem('user', JSON.stringify(user));
      expect(safeGetUser()).toEqual(user);
    });

    it('should return null when user is not set', () => {
      expect(safeGetUser()).toBeNull();
    });

    it('should return null and clean up when user is "undefined" string', () => {
      localStorage.setItem('user', 'undefined');
      expect(safeGetUser()).toBeNull();
      expect(localStorage.getItem('user')).toBeNull();
    });

    it('should return null and clean up when user is "null" string', () => {
      localStorage.setItem('user', 'null');
      expect(safeGetUser()).toBeNull();
      expect(localStorage.getItem('user')).toBeNull();
    });

    it('should return null and clean up when data is corrupted', () => {
      localStorage.setItem('user', '{invalid json}');
      expect(safeGetUser()).toBeNull();
      expect(localStorage.getItem('user')).toBeNull();
    });
  });

  describe('clearAuthData', () => {
    it('should remove both token and user', () => {
      localStorage.setItem('token', 'test-token');
      localStorage.setItem('user', JSON.stringify({ email: 'test@example.com' }));
      clearAuthData();
      expect(localStorage.getItem('token')).toBeNull();
      expect(localStorage.getItem('user')).toBeNull();
    });
  });
});
