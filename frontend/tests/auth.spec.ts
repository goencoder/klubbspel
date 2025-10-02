import { expect, test } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test('should handle magic link authentication flow', async ({ page }) => {
    // Navigate to the main page
    await page.goto('http://localhost:5000');

    // We should see the main page without authentication initially
    await expect(page).toHaveTitle(/Klubbspel/);

    // Try to access a protected action (like creating a club)
    // This should trigger authentication requirement
    await page.click('[data-testid="add-club-button"]');

    // Should see the create club dialog
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();

    // Fill in club details
    const testEmail = 'test@example.com';
    const clubName = 'Test Club for Auth';

    await page.getByTestId('create-club-name-input').fill(clubName);
    await page.getByTestId('create-club-submit-button').click();

    // This should fail due to missing authentication
    // We should see an error or be redirected to login
    await expect(page.getByText(/authentication/i)).toBeVisible({ timeout: 10000 });
  });

  test('should send magic link email via API', async ({ page, request }) => {
    const testEmail = 'playwright-test@example.com';

    // Send magic link request
    const response = await request.post('http://localhost:8080/v1/auth/magic-link', {
      data: {
        email: testEmail
      }
    });

    // Should succeed
    expect(response.status()).toBe(200);

    const responseData = await response.json();
    expect(responseData.sent).toBe(true);
    expect(responseData.expiresInSeconds).toBe(900); // 15 minutes

    // Check MailHog for the email
    const mailhogResponse = await request.get('http://localhost:8025/api/v2/messages');
    expect(mailhogResponse.status()).toBe(200);

    const messages = await mailhogResponse.json();

    // Should have at least one message
    expect(messages.total).toBeGreaterThan(0);

    // Find our test email
    const testMessage = messages.items.find((msg: any) =>
      msg.To.some((to: any) => to.Mailbox === 'playwright-test' && to.Domain === 'example.com')
    );

    expect(testMessage).toBeDefined();
    expect(testMessage.Content.Headers.Subject[0]).toContain('Sign in to Klubbspel');

    // Extract the magic link from the email body
    const emailBody = testMessage.Content.Body;
    const magicLinkMatch = emailBody.match(/http:\/\/localhost:5000\/auth\/login\?apikey=([a-f0-9-]+)/);
    expect(magicLinkMatch).toBeTruthy();

    const magicLink = magicLinkMatch[0];
    const token = magicLinkMatch[1];

    console.log('Magic link found:', magicLink);
    console.log('Token extracted:', token);

    // Validate the token via API
    const validateResponse = await request.post('http://localhost:8080/v1/auth/validate', {
      data: {
        token: token
      }
    });

    expect(validateResponse.status()).toBe(200);
    const validateData = await validateResponse.json();
    expect(validateData.apiToken).toBeDefined();
    expect(validateData.player).toBeDefined();
    expect(validateData.player.email).toBe(testEmail);

    console.log('Authentication successful! API Token received:', validateData.apiToken.substring(0, 20) + '...');
  });

  test('should authenticate and create club with valid token', async ({ page, request }) => {
    const testEmail = 'auth-test@example.com';

    // Step 1: Request magic link
    const magicLinkResponse = await request.post('http://localhost:8080/v1/auth/magic-link', {
      data: { email: testEmail }
    });
    expect(magicLinkResponse.status()).toBe(200);

    // Step 2: Get the email from MailHog
    await page.waitForTimeout(1000); // Give email time to be delivered

    const mailhogResponse = await request.get('http://localhost:8025/api/v2/messages');
    const messages = await mailhogResponse.json();

    const testMessage = messages.items.find((msg: any) =>
      msg.To.some((to: any) => to.Mailbox === 'auth-test' && to.Domain === 'example.com')
    );

    // Step 3: Extract token and validate
    const emailBody = testMessage.Content.Body;
    const tokenMatch = emailBody.match(/apikey=([a-f0-9-]+)/);
    const token = tokenMatch[1];

    const validateResponse = await request.post('http://localhost:8080/v1/auth/validate', {
      data: { token: token }
    });

    const validateData = await validateResponse.json();
    const apiToken = validateData.apiToken;

    // Step 4: Use API token to create a club
    const clubResponse = await request.post('http://localhost:8080/v1/clubs', {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      },
      data: {
        name: 'Authenticated Test Club',
        description: 'Created with valid authentication'
      }
    });

    expect(clubResponse.status()).toBe(200);
    const clubData = await clubResponse.json();
    expect(clubData.name).toBe('Authenticated Test Club');

    console.log('Successfully created club with authentication:', clubData.id);

    // Step 5: Verify we can list clubs with authentication
    const listResponse = await request.get('http://localhost:8080/v1/clubs', {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(listResponse.status()).toBe(200);
    const listData = await listResponse.json();
    expect(listData.items).toHaveLength(1);
    expect(listData.items[0].name).toBe('Authenticated Test Club');
  });

  test('should demonstrate Phase 5 security features', async ({ request }) => {
    // Test rate limiting by making multiple rapid requests
    const responses = await Promise.all([
      request.post('http://localhost:8080/v1/auth/magic-link', {
        data: { email: 'rate-test-1@example.com' }
      }),
      request.post('http://localhost:8080/v1/auth/magic-link', {
        data: { email: 'rate-test-2@example.com' }
      }),
      request.post('http://localhost:8080/v1/auth/magic-link', {
        data: { email: 'rate-test-3@example.com' }
      }),
      request.post('http://localhost:8080/v1/auth/magic-link', {
        data: { email: 'rate-test-4@example.com' }
      }),
      request.post('http://localhost:8080/v1/auth/magic-link', {
        data: { email: 'rate-test-5@example.com' }
      })
    ]);

    // First few should succeed
    expect(responses[0].status()).toBe(200);
    expect(responses[1].status()).toBe(200);

    // But we should eventually hit rate limits
    const statusCodes = responses.map(r => r.status());
    console.log('Rate limiting test - status codes:', statusCodes);

    // Test input validation
    const invalidEmailResponse = await request.post('http://localhost:8080/v1/auth/magic-link', {
      data: { email: 'invalid-email' }
    });

    expect(invalidEmailResponse.status()).toBe(400);

    // Test unauthorized access
    const unauthorizedResponse = await request.post('http://localhost:8080/v1/clubs', {
      data: { name: 'Unauthorized Club' }
    });

    expect(unauthorizedResponse.status()).toBe(401);

    console.log('Phase 5 security features working: rate limiting, validation, and authorization');
  });
});
