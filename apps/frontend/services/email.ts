import { Resend } from 'resend';
import handlebars from 'handlebars';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Initialize Resend
const resend = new Resend(process.env.RESEND_API_KEY);
const fromEmail = process.env.RESEND_FROM_EMAIL || 'noreply@aifastlane.net';

// Load and compile email templates
const templatesDir = path.join(__dirname, '../templates');
const verificationTemplate = handlebars.compile(
  fs.readFileSync(path.join(templatesDir, 'verification-email.hbs'), 'utf8')
);

export interface VerificationEmailData {
  email: string;
  verificationCode: string;
  appName?: string;
  expirationMinutes?: number;
}

export interface EmailResult {
  success: boolean;
  emailId?: string;
  error?: string;
}

export async function sendVerificationEmail(data: VerificationEmailData): Promise<EmailResult> {
  try {
    const html = verificationTemplate({
      appName: data.appName || 'AI Fastlane',
      verificationCode: data.verificationCode,
      expirationMinutes: data.expirationMinutes || 10
    });

    const result = await resend.emails.send({
      from: fromEmail,
      to: [data.email],
      subject: `Verify your email - ${data.appName || 'AI Fastlane'}`,
      html
    });

    if (result.error) {
      return {
        success: false,
        error: result.error.message || 'Failed to send email'
      };
    }

    return {
      success: true,
      emailId: result.data?.id
    };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

