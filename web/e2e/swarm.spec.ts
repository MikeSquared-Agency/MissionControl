import { test, expect } from '@playwright/test'

test.describe('Swarm Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('should show swarm tab', async ({ page }) => {
    await page.waitForLoadState('networkidle')

    // The Swarm tab button should be visible on page load.
    const swarmTab = page.locator('button:has-text("Swarm")')
    await expect(swarmTab).toBeVisible({ timeout: 5000 })
  })

  test('should switch to swarm view', async ({ page }) => {
    await page.waitForLoadState('networkidle')

    // Click the Swarm tab.
    await page.locator('button:has-text("Swarm")').click()

    // Live and Schedule sub-tabs should appear.
    const liveTab = page.locator('button:has-text("Live")')
    const scheduleTab = page.locator('button:has-text("Schedule")')

    await expect(liveTab).toBeVisible({ timeout: 5000 })
    await expect(scheduleTab).toBeVisible({ timeout: 5000 })
  })

  test('should show schedule placeholder', async ({ page }) => {
    await page.waitForLoadState('networkidle')

    // Navigate to Swarm > Schedule.
    await page.locator('button:has-text("Swarm")').click()
    await page.locator('button:has-text("Schedule")').click()

    // The placeholder text should be visible.
    const placeholder = page.locator('text=/Phase 2/i')
    await expect(placeholder).toBeVisible({ timeout: 5000 })
  })

  test('should switch back to live', async ({ page }) => {
    await page.waitForLoadState('networkidle')

    // Navigate to Swarm > Schedule, then back to Live.
    await page.locator('button:has-text("Swarm")').click()
    await page.locator('button:has-text("Schedule")').click()
    await page.locator('button:has-text("Live")').click()

    // Schedule placeholder should disappear.
    const placeholder = page.locator('text=/Phase 2/i')
    await expect(placeholder).not.toBeVisible({ timeout: 5000 })

    // Live or Schedule button with active style should indicate Live is selected.
    const liveTab = page.locator('button:has-text("Live")')
    await expect(liveTab).toBeVisible()
  })
})
