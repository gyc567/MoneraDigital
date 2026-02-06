import { Resend } from 'resend';
import logger from './logger.js';

const RESEND_API_KEY = import.meta.env.VITE_RESEND_API_KEY;
const APP_URL = import.meta.env.VITE_APP_URL || 'https://monera-digital.vercel.app';
const SENDER_EMAIL = import.meta.env.VITE_SENDER_EMAIL || 'noreply@monera-digital.app';

// Initialize Resend client
const resend = RESEND_API_KEY ? new Resend(RESEND_API_KEY) : null;

export class EmailService {
  /**
   * Send address verification email
   */
  static async sendVerificationEmail(email: string, verificationToken: string): Promise<void> {
    if (!resend) {
      logger.warn('Resend API not configured, skipping email');
      // In development, just log the token
      logger.info({ email, verificationToken }, 'Verification email would be sent (dev mode)');
      return;
    }

    try {
      const verificationLink = `${APP_URL}/verify-address?token=${encodeURIComponent(verificationToken)}`;

      logger.info({ email }, 'Sending address verification email');

      await resend.emails.send({
        from: SENDER_EMAIL,
        to: email,
        subject: 'Verify Your Withdrawal Address - MoneraDigital',
        html: this.getVerificationEmailHTML(verificationLink, verificationToken),
      });

      logger.info({ email }, 'Verification email sent successfully');
    } catch (error: any) {
      logger.error({ error: error.message, email }, 'Failed to send verification email');
      throw new Error('Failed to send verification email');
    }
  }

  /**
   * Send withdrawal confirmation email
   */
  static async sendWithdrawalConfirmation(
    email: string,
    withdrawalDetails: {
      amount: string;
      asset: string;
      toAddress: string;
      status: string;
      txHash?: string;
    }
  ): Promise<void> {
    if (!resend) {
      logger.warn('Resend API not configured, skipping email');
      logger.info({ email, details: withdrawalDetails }, 'Withdrawal confirmation email would be sent (dev mode)');
      return;
    }

    try {
      logger.info({ email }, 'Sending withdrawal confirmation email');

      await resend.emails.send({
        from: SENDER_EMAIL,
        to: email,
        subject: `Withdrawal Confirmation - ${withdrawalDetails.amount} ${withdrawalDetails.asset}`,
        html: this.getWithdrawalConfirmationHTML(withdrawalDetails),
      });

      logger.info({ email }, 'Withdrawal confirmation email sent successfully');
    } catch (error: any) {
      logger.error({ error: error.message, email }, 'Failed to send withdrawal confirmation email');
      throw new Error('Failed to send withdrawal confirmation email');
    }
  }

  /**
   * HTML template for verification email
   */
  private static getVerificationEmailHTML(verificationLink: string, token: string): string {
    return `
      <!DOCTYPE html>
      <html lang="en">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <style>
          body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; }
          .container { max-width: 600px; margin: 0 auto; padding: 20px; }
          .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
          .content { background: #f8f9fa; padding: 30px; border-radius: 0 0 8px 8px; }
          .button { display: inline-block; background: #667eea; color: white; padding: 12px 30px; border-radius: 4px; text-decoration: none; margin: 20px 0; }
          .footer { text-align: center; font-size: 12px; color: #999; margin-top: 20px; }
          .token-box { background: white; padding: 15px; border-radius: 4px; font-family: monospace; border: 1px solid #e0e0e0; margin: 15px 0; }
        </style>
      </head>
      <body>
        <div class="container">
          <div class="header">
            <h1>Address Verification</h1>
            <p>MoneraDigital - Secure Your Withdrawals</p>
          </div>
          <div class="content">
            <p>Hello,</p>
            <p>You've requested to add a new withdrawal address to your MoneraDigital account. To proceed, please verify this address by clicking the button below:</p>

            <div style="text-align: center;">
              <a href="${verificationLink}" class="button">Verify Address</a>
            </div>

            <p>Or copy and paste this verification code in the app:</p>
            <div class="token-box">${token}</div>

            <p style="color: #999; font-size: 14px;">This link will expire in 24 hours for security reasons.</p>

            <p>If you didn't request this, please ignore this email.</p>

            <p>Best regards,<br>The MoneraDigital Team</p>
          </div>
          <div class="footer">
            <p>© 2024 MoneraDigital. All rights reserved.</p>
          </div>
        </div>
      </body>
      </html>
    `;
  }

  /**
   * HTML template for withdrawal confirmation email
   */
  private static getWithdrawalConfirmationHTML(details: {
    amount: string;
    asset: string;
    toAddress: string;
    status: string;
    txHash?: string;
  }): string {
    const statusColor = details.status === 'COMPLETED' ? '#10b981' : details.status === 'FAILED' ? '#ef4444' : '#f59e0b';

    return `
      <!DOCTYPE html>
      <html lang="en">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <style>
          body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; }
          .container { max-width: 600px; margin: 0 auto; padding: 20px; }
          .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
          .content { background: #f8f9fa; padding: 30px; border-radius: 0 0 8px 8px; }
          .status-badge { display: inline-block; background: ${statusColor}; color: white; padding: 8px 16px; border-radius: 4px; font-weight: bold; }
          .details-box { background: white; padding: 20px; border-radius: 4px; margin: 20px 0; border-left: 4px solid ${statusColor}; }
          .detail-row { display: flex; justify-content: space-between; margin: 10px 0; }
          .detail-label { color: #666; font-weight: 500; }
          .detail-value { color: #333; font-weight: bold; }
          .footer { text-align: center; font-size: 12px; color: #999; margin-top: 20px; }
          .tx-hash { font-family: monospace; font-size: 12px; word-break: break-all; }
        </style>
      </head>
      <body>
        <div class="container">
          <div class="header">
            <h1>Withdrawal Confirmation</h1>
            <p>MoneraDigital</p>
          </div>
          <div class="content">
            <p>Hello,</p>
            <p>Your withdrawal request has been processed. Here are the details:</p>

            <div class="details-box">
              <div style="text-align: center; margin-bottom: 15px;">
                <span class="status-badge">${details.status}</span>
              </div>
              <div class="detail-row">
                <span class="detail-label">Amount:</span>
                <span class="detail-value">${details.amount} ${details.asset}</span>
              </div>
              <div class="detail-row">
                <span class="detail-label">Destination Address:</span>
                <span class="detail-value" style="font-size: 12px; word-break: break-all;">${details.toAddress}</span>
              </div>
              ${details.txHash ? `<div class="detail-row">
                <span class="detail-label">Transaction Hash:</span>
                <span class="detail-value tx-hash">${details.txHash}</span>
              </div>` : ''}
            </div>

            <p style="color: #999; font-size: 14px;">Your withdrawal status: <strong>${details.status}</strong></p>

            <p>If you have any questions, please contact our support team.</p>

            <p>Best regards,<br>The MoneraDigital Team</p>
          </div>
          <div class="footer">
            <p>© 2024 MoneraDigital. All rights reserved.</p>
          </div>
        </div>
      </body>
      </html>
    `;
  }
}
