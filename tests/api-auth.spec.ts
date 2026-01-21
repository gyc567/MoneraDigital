import { test, expect } from '@playwright/test';

test.describe('Authentication API', () => {
  const baseURL = 'http://127.0.0.1:8081';

  // Helper function to reduce duplication (KISS principle)
  const registerUser = async (request, email, password) => {
    return request.post(`${baseURL}/api/auth/register`, {
      data: { email, password },
    });
  };

  test('should register a new user successfully', async ({ request }) => {
    const email = `test.api.${Date.now()}@example.com`;
    const password = 'Password123!';

    const response = await registerUser(request, email, password);

    expect(response.status()).toBe(201);
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody).toHaveProperty('email', email);
    expect(typeof responseBody.id).toBe('number');
  });

  test('should fail to register a user with a duplicate email', async ({ request }) => {
    const email = `test.api.duplicate.${Date.now()}@example.com`;
    const password = 'Password123!';

    // First registration should succeed
    const firstResponse = await registerUser(request, email, password);
    expect(firstResponse.status()).toBe(201);

    // Second registration with the same email should fail
    const secondResponse = await registerUser(request, email, password);
    expect(secondResponse.status()).toBe(400);
    const responseBody = await secondResponse.json();
    expect(responseBody).toHaveProperty('error', 'email already registered');
  });

  test('should fail to register with an invalid email', async ({ request }) => {
    const response = await registerUser(request, 'not-an-email', 'Password123!');
    
    expect(response.status()).toBe(400);
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('error', 'validation error on field \'email\': invalid email format');
  });

  test('should fail to register with a weak password', async ({ request }) => {
    const email = `test.api.weakpass.${Date.now()}@example.com`;
    const response = await registerUser(request, email, '123');

    expect(response.status()).toBe(400);
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('error', 'validation error on field \'password\': password must be at least 8 characters');
  });
});
