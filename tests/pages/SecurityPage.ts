import { Page, Locator } from '@playwright/test';

/**
 * Page Object Model for Security Dashboard
 * Handles 2FA setup, enable, disable flows
 */
export class SecurityPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly enable2FAButton: Locator;
  readonly disable2FAButton: Locator;
  readonly statusIndicator: Locator;
  readonly setupDialog: Locator;
  readonly disableDialog: Locator;
  readonly qrCodeImage: Locator;
  readonly secretKeyCode: Locator;
  readonly copySecretButton: Locator;
  readonly nextBackupCodesButton: Locator;
  readonly backupCodesGrid: Locator;
  readonly verificationInput: Locator;
  readonly verifyButton: Locator;
  readonly disableTokenInput: Locator;
  readonly disableConfirmButton: Locator;

  constructor(page: Page) {
    this.page = page;

    // Page elements
    this.heading = page.getByRole('heading', { name: /安全设置|Security/i });
    this.enable2FAButton = page.getByRole('button', { name: /启用2FA|Enable 2FA/i });
    this.disable2FAButton = page.getByRole('button', { name: /禁用2FA|Disable 2FA/i });
    this.statusIndicator = page.getByText(/状态|Status/i);

    // Setup dialog elements
    this.setupDialog = page.getByRole('dialog');
    this.qrCodeImage = page.locator('img[alt*="2FA QR Code"]');
    this.secretKeyCode = page.locator('code').filter({ hasText: /^[A-Z0-9]{32}$/i });
    this.copySecretButton = page.getByRole('button').filter({ has: page.locator('svg') });
    this.nextBackupCodesButton = page.getByRole('button', { name: /下一步|Next|备份码|Backup/i });
    this.backupCodesGrid = page.locator('.grid').filter({ has: page.locator('code') });
    this.verificationInput = page.locator('input[placeholder="000000"]');
    this.verifyButton = page.getByRole('button', { name: /验证|Verify/i });

    // Disable dialog elements
    this.disableDialog = page.getByRole('dialog');
    this.disableTokenInput = page.locator('input[placeholder="000000"]').last();
    this.disableConfirmButton = page.getByRole('button', { name: /禁用2FA|Disable 2FA/i }).last();
  }

  async goto() {
    await this.page.goto('/dashboard/security');
    await this.page.waitForLoadState('networkidle');
  }

  async clickEnable2FA() {
    await this.enable2FAButton.click();
    await this.page.waitForResponse(resp =>
      resp.url().includes('/api/auth/2fa/setup') && resp.status() === 200
    );
  }

  async waitForQRCode() {
    await this.qrCodeImage.waitFor({ state: 'visible', timeout: 10000 });
  }

  async getSecretKey(): Promise<string> {
    await this.secretKeyCode.waitFor({ state: 'visible' });
    const secret = await this.secretKeyCode.textContent();
    return secret?.trim() || '';
  }

  async clickNextToBackupCodes() {
    await this.nextBackupCodesButton.click();
    await this.backupCodesGrid.waitFor({ state: 'visible' });
  }

  async getBackupCodes(): Promise<string[]> {
    const codeElements = await this.backupCodesGrid.locator('code').all();
    const codes: string[] = [];
    for (const element of codeElements) {
      const text = await element.textContent();
      if (text) codes.push(text.trim());
    }
    return codes;
  }

  async enterVerificationCode(code: string) {
    await this.verificationInput.fill(code);
  }

  async clickVerify() {
    await this.verifyButton.click();
    await this.page.waitForResponse(resp =>
      resp.url().includes('/api/auth/2fa/enable')
    );
  }

  async is2FAEnabled(): Promise<boolean> {
    const statusText = await this.statusIndicator.textContent();
    return statusText?.includes('已启用') || statusText?.includes('Enabled') || false;
  }

  async clickDisable2FA() {
    await this.disable2FAButton.click();
    await this.disableDialog.waitFor({ state: 'visible' });
  }

  async enterDisableToken(token: string) {
    await this.disableTokenInput.fill(token);
  }

  async confirmDisable() {
    await this.disableConfirmButton.click();
    await this.page.waitForResponse(resp =>
      resp.url().includes('/api/auth/2fa/disable')
    );
  }

  async takeScreenshot(name: string) {
    await this.page.screenshot({
      path: `playwright-report/screenshots/${name}.png`,
      fullPage: true
    });
  }
}
