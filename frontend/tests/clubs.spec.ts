import { test, expect } from '@playwright/test';

test.describe('Club Management', () => {
  test('should create a new club', async ({ page }) => {
    // Navigate to the clubs page
    await page.goto('/clubs');
    
    // Wait for the page to load
    await expect(page.getByTestId('clubs-title')).toBeVisible();
    
    // Click the "Create New Club" button
    await page.getByTestId('create-club-button').click();
    
    // Wait for the dialog to open
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();
    
    // Fill in the club name
    const clubName = `Test Club ${Date.now()}`;
    await page.getByTestId('create-club-name-input').fill(clubName);
    
    // Submit the form
    await page.getByTestId('create-club-submit-button').click();
    
    // Wait for the dialog to close
    await expect(page.getByTestId('create-club-dialog')).not.toBeVisible();
    
    // Verify the club appears in the list
    await expect(page.getByText(clubName)).toBeVisible();
  });

  test('should validate required club name', async ({ page }) => {
    // Navigate to the clubs page
    await page.goto('/clubs');
    
    // Click the "Create New Club" button
    await page.getByTestId('create-club-button').click();
    
    // Wait for the dialog to open
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();
    
    // The button should be disabled when name is empty
    await expect(page.getByTestId('create-club-submit-button')).toBeDisabled();
  });

  test('should validate minimum club name length', async ({ page }) => {
    // Navigate to the clubs page
    await page.goto('/clubs');
    
    // Click the "Create New Club" button
    await page.getByTestId('create-club-button').click();
    
    // Wait for the dialog to open
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();
    
    // Fill in a name that's too short (1 character)
    await page.getByTestId('create-club-name-input').fill('A');
    
    // Submit the form
    await page.getByTestId('create-club-submit-button').click();
    
    // Should show validation error - we'll check for any error message
    await expect(page.locator('body')).toContainText(/minimum|min|kort/i, { timeout: 5000 });
  });

  test('should search for clubs', async ({ page }) => {
    // Navigate to the clubs page
    await page.goto('/clubs');
    
    // Wait for the page to load
    await expect(page.getByTestId('clubs-title')).toBeVisible();
    
    // First create a club to search for
    await page.getByTestId('create-club-button').click();
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();
    
    const clubName = `Searchable Club ${Date.now()}`;
    await page.getByTestId('create-club-name-input').fill(clubName);
    await page.getByTestId('create-club-submit-button').click();
    await expect(page.getByTestId('create-club-dialog')).not.toBeVisible();
    
    // Now test search functionality
    const searchInput = page.getByTestId('search-clubs-input');
    await searchInput.fill('Searchable');
    
    // Wait for search results
    await page.waitForTimeout(500); // Wait for debounced search
    
    // Should show the club we just created
    await expect(page.getByText(clubName)).toBeVisible();
    
    // Search for something that doesn't exist
    await searchInput.fill('NonExistentClub');
    await page.waitForTimeout(500); // Wait for debounced search
    
    // The club we created should not be visible anymore
    await expect(page.getByText(clubName)).not.toBeVisible();
  });

  test('should edit an existing club', async ({ page }) => {
    // Navigate to the clubs page
    await page.goto('/clubs');
    
    // Create a club first
    await page.getByTestId('create-club-button').click();
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();
    
    const originalName = `Editable Club ${Date.now()}`;
    await page.getByTestId('create-club-name-input').fill(originalName);
    await page.getByTestId('create-club-submit-button').click();
    await expect(page.getByTestId('create-club-dialog')).not.toBeVisible();
    
    // Wait for the club to appear and get its ID from the DOM
    await expect(page.getByText(originalName)).toBeVisible();
    
    // Find the club card and click edit (we'll use a more generic approach since we don't know the ID yet)
    const clubCard = page.locator(`[data-testid^="club-card-"]`).filter({ hasText: originalName });
    await clubCard.locator(`[data-testid^="edit-club-"]`).click();
    
    // Wait for edit dialog
    await expect(page.getByTestId('edit-club-dialog')).toBeVisible();
    
    // Update the name
    const newName = `Updated Club ${Date.now()}`;
    await page.getByTestId('edit-club-name-input').clear();
    await page.getByTestId('edit-club-name-input').fill(newName);
    
    // Save changes
    await page.getByTestId('edit-club-submit-button').click();
    await expect(page.getByTestId('edit-club-dialog')).not.toBeVisible();
    
    // Verify the updated name appears
    await expect(page.getByText(newName)).toBeVisible();
    await expect(page.getByText(originalName)).not.toBeVisible();
  });

  test('should delete a club', async ({ page }) => {
    // Navigate to the clubs page
    await page.goto('/clubs');
    
    // Create a club first
    await page.getByTestId('create-club-button').click();
    await expect(page.getByTestId('create-club-dialog')).toBeVisible();
    
    const clubName = `Deletable Club ${Date.now()}`;
    await page.getByTestId('create-club-name-input').fill(clubName);
    await page.getByTestId('create-club-submit-button').click();
    await expect(page.getByTestId('create-club-dialog')).not.toBeVisible();
    
    // Wait for the club to appear
    await expect(page.getByText(clubName)).toBeVisible();
    
    // Find the club card and click delete
    const clubCard = page.locator(`[data-testid^="club-card-"]`).filter({ hasText: clubName });
    await clubCard.locator(`[data-testid^="delete-club-"]`).click();
    
    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible();
    
    // Click the confirm delete button (we'll use a more generic approach)
    await page.getByRole('button', { name: /delete|ta bort/i }).last().click();
    
    // Verify the club is removed
    await expect(page.getByText(clubName)).not.toBeVisible();
  });
});
