import { test, expect } from '@playwright/test';

test.describe('Complete User Journey - Authentication to Match Reporting', () => {
  test('should complete full user workflow from login to leaderboard verification', async ({ page }) => {
    // Step 1: Navigate to login page
    await page.goto('http://localhost:5000/login');
    
    // Step 2: Check if already logged in, if so log out first
    const debugMenuButton = page.getByRole('button', { name: /debug manual|Anna Testsson|test@example\.com/i });
    if (await debugMenuButton.isVisible()) {
      await debugMenuButton.click();
      await page.getByRole('menuitem', { name: 'Logga ut' }).click();
      await page.goto('http://localhost:5000/login');
    }

    // Step 3: Send magic link
    const emailInput = page.getByRole('textbox', { name: 'E-postadress' });
    await emailInput.fill('test@example.com');
    await page.getByRole('button', { name: 'Skicka magisk länk' }).click();
    
    // Verify magic link sent notification
    await expect(page.getByText('Magisk länk skickad!')).toBeVisible();

    // Step 4: Open MailHog and follow magic link
    await page.goto('http://localhost:8025/#');
    
    // Wait for and click on the email
    await page.getByText('Klubbspel').first().click();
    
    // Click the magic login link
    const magicLink = page.getByRole('link', { name: /http:\/\/localhost:5000\/auth\// });
    await magicLink.click();
    
    // Verify successful login
    await expect(page.getByText('Välkommen! Du har loggats in framgångsrikt.')).toBeVisible();

    // Step 5: Go to settings and fill out profile
    const userMenuButton = page.getByRole('button', { name: 'test@example.com' });
    await userMenuButton.click();
    await page.getByRole('menuitem', { name: 'Inställningar' }).click();

    // Fill out required profile fields
    await page.getByRole('textbox', { name: 'Förnamn *' }).fill('Anna');
    await page.getByRole('textbox', { name: 'Efternamn *' }).fill('Testsson');
    await page.getByRole('button', { name: 'Spara' }).click();
    
    // Verify profile updated
    await expect(page.getByText('Profilen har uppdaterats')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Anna Testsson' })).toBeVisible();

    // Step 6: Create a new club
    await page.getByRole('link', { name: 'Klubbar' }).click();
    await page.getByRole('button', { name: 'Skapa ny klubb' }).click();
    
    await page.getByRole('textbox', { name: 'Klubbnamn' }).fill('Playwright Test Club');
    await page.getByRole('button', { name: 'Skapa' }).click();
    
    // Verify club created
    await expect(page.getByText('Klubb skapad framgångsrikt')).toBeVisible();
    
    // Navigate to club details
    const clubLink = page.getByRole('link', { name: /Playwright Test Club.*Visa detaljer/ });
    await clubLink.getByRole('button', { name: 'Visa detaljer' }).click();

    // Step 7: Add players to the club
    await page.getByRole('tab', { name: 'Medlemmar' }).click();
    
    // Add first player (Erik)
    await page.getByRole('button', { name: 'Lägg till spelare' }).click();
    await page.getByRole('textbox', { name: 'Förnamn *' }).fill('Erik');
    await page.getByRole('textbox', { name: 'Efternamn *' }).fill('Svensson');
    await page.getByRole('textbox', { name: 'E-post (valfritt)' }).fill('erik.svensson@example.com');
    await page.getByRole('button', { name: 'Lägg till spelare' }).click();
    
    // Verify first player added
    await expect(page.getByText('Spelare tillagd')).toBeVisible();
    
    // Add second player (Maria)
    await page.getByRole('button', { name: 'Lägg till spelare' }).click();
    await page.getByRole('textbox', { name: 'Förnamn *' }).fill('Maria');
    await page.getByRole('textbox', { name: 'Efternamn *' }).fill('Andersson');
    await page.getByRole('textbox', { name: 'E-post (valfritt)' }).fill('maria.andersson@example.com');
    await page.getByRole('button', { name: 'Lägg till spelare' }).click();
    
    // Verify second player added and total count
    await expect(page.getByText('Spelare tillagd')).toBeVisible();
    await expect(page.getByText('(3 totalt)')).toBeVisible();

    // Step 8: Create tournament series
    await page.getByRole('tab', { name: 'Serier' }).click();
    await page.getByRole('button', { name: 'Skapa ny serie' }).click();
    
    // Fill series form
    await page.getByRole('textbox', { name: 'Ange serienamn' }).fill('Playwright Championship 2025');
    await page.locator('#series-start-date-input').fill('2025-09-24T09:00');
    await page.locator('#series-end-date-input').fill('2025-10-01T18:00');
    await page.getByRole('button', { name: 'Skapa' }).click();
    
    // Verify series created and navigate to it
    await expect(page.getByText('Serie skapad framgångsrikt')).toBeVisible();
    await page.getByRole('link', { name: 'View' }).click();

    // Step 9: Report match results
    await page.getByRole('button', { name: 'Rapportera matcher' }).click();
    
    // Select Player A (Erik Johansson)
    await page.locator('#player-selector-trigger-match-player-a-selector').click();
    await page.getByPlaceholder('Sök...').fill('Erik');
    await page.getByRole('option', { name: 'Erik Johansson' }).click();
    
    // Select Player B (Anna Andersson)  
    await page.getByText('Välj spelare').click();
    await page.getByPlaceholder('Sök...').fill('Anna');
    await page.getByRole('option', { name: 'Anna Andersson' }).click();
    
    // Set scores (Erik wins 3-1)
    await page.getByRole('spinbutton', { name: 'Spelare A poäng' }).fill('3');
    await page.getByRole('spinbutton', { name: 'Spelare B poäng' }).fill('1');
    
    // Submit match result
    await page.getByRole('button', { name: 'Rapportera matcher' }).click();
    
    // Verify match reported
    await expect(page.getByText('Match rapporterad framgångsrikt')).toBeVisible();
    
    // Verify match appears in table
    await expect(page.getByRole('cell', { name: 'Erik Johansson' })).toBeVisible();
    await expect(page.getByRole('cell', { name: '3 - 1' })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Anna Andersson' })).toBeVisible();

    // Step 10: Verify leaderboard updates
    await page.getByRole('tab', { name: 'Resultattabell' }).click();
    
    // Verify Erik is #1 with correct stats
    await expect(page.getByRole('cell', { name: '#1' })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Erik Johansson' })).toBeVisible();
    await expect(page.getByRole('cell', { name: '1016' })).toBeVisible(); // ELO rating
    await expect(page.getByRole('cell', { name: '100%' })).toBeVisible(); // Win percentage
    
    // Verify Anna is #2 with correct stats
    await expect(page.getByRole('cell', { name: '#2' })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Anna Andersson' })).toBeVisible();
    await expect(page.getByRole('cell', { name: '984' })).toBeVisible(); // ELO rating
    await expect(page.getByRole('cell', { name: '0%' })).toBeVisible(); // Win percentage
  });
});