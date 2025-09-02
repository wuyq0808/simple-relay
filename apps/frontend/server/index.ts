import 'dotenv/config';
import express, { Request, Response, NextFunction } from 'express';
import cors from 'cors';
import cookieParser from 'cookie-parser';
import path from 'path';
import { fileURLToPath } from 'url';
import { readFileSync } from 'fs';
import rateLimit from 'express-rate-limit';
import validator from 'validator';
import { sendVerificationEmail } from '../services/email.js';
import { UserDatabase } from '../services/database.js';
import { ConfigService } from '../services/config.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.PORT || 3000;

app.set('trust proxy', 1);

const ipRateLimit = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 30, // 30 requests per hour per IP
  message: 'Too many requests from this IP. Try again later.',
});

app.use(cors());
app.use(cookieParser(process.env.COOKIE_SECRET));
app.use(express.json());
app.use(express.static(path.join(process.cwd(), 'dist')));


function generateVerificationCode(): string {
  return Math.floor(100000 + Math.random() * 900000).toString();
}

function setLoginCookie(res: Response, email: string): void {
  res.cookie('user_email', email, {
    httpOnly: true,
    secure: process.env.DEPLOYMENT_ENV !== 'development',
    sameSite: 'lax',
    signed: true
  });
}

function requireAuth(req: Request, res: Response, next: NextFunction): void {
  const email = req.signedCookies.user_email;
  
  if (!email) {
    res.status(401).json({ error: 'Authentication required' });
    return;
  }
  
  (req as any).userEmail = email;
  next();
}

app.get('/api/health', (_req: Request, res: Response) => {
  res.json({ status: 'ok' });
});

app.post('/api/signin', ipRateLimit, async (req, res) => {
  const { email } = req.body;
  
  if (!email || !validator.isEmail(email)) {
    return res.status(400).json({ error: 'Valid email address is required' });
  }

  // Check if signup is enabled
  try {
    const signupEnabled = await ConfigService.getConfig('signup_enabled');
    if (signupEnabled === false) {
      return res.status(403).json({ error: 'Sign up is currently disabled' });
    }
  } catch (error) {
    console.error('Error checking signup config:', error);
    return res.status(500).json({ error: 'Configuration service unavailable' });
  }

  const existingUser = await UserDatabase.findByEmail(email);
  
  const verificationCode = generateVerificationCode();
  const verificationExpiresAt = new Date(Date.now() + 10 * 60 * 1000);

  if (existingUser) {
    await UserDatabase.updateUser(email, {
      verification_token: verificationCode,
      verification_expires_at: verificationExpiresAt
    });
  } else {
    await UserDatabase.create({
      email,
      last_login: null,
      verification_token: verificationCode,
      verification_expires_at: verificationExpiresAt
    });
  }

  try {
    const result = await sendVerificationEmail({
      email,
      verificationCode,
      appName: 'AI Fastlane',
      expirationMinutes: 10
    });

    if (result.success) {
      res.json({ 
        message: 'Verification email sent successfully',
        user: { email }
      });
    } else {
      res.status(500).json({ error: result.error || 'Failed to send verification email' });
    }
  } catch (error) {
    console.error('Email sending error:', error);
    res.status(500).json({ error: 'Failed to send verification email' });
  }
});

app.post('/api/verify', async (req, res) => {
  const { email, code } = req.body;
  
  if (!email || !validator.isEmail(email) || !code) {
    return res.status(400).json({ error: 'Valid email and verification code are required' });
  }

  const user = await UserDatabase.findByEmail(email);
  
  if (!user) {
    return res.status(400).json({ error: 'No verification code found for this email' });
  }

  if (!user.verification_token) {
    return res.status(400).json({ error: 'No verification code found for this email' });
  }

  if (!UserDatabase.isVerificationTokenValid(user)) {
    return res.status(400).json({ error: 'Verification code has expired' });
  }

  if (user.verification_token !== code) {
    return res.status(400).json({ error: 'Invalid verification code' });
  }

  await UserDatabase.verifyUser(email);
  await UserDatabase.updateLastLogin(email);
  
  setLoginCookie(res, email);
  
  console.log(`User verified: ${email}`);
  
  res.json({ message: 'Email verified successfully', email });
});

app.get('/api/profile', requireAuth, async (req, res) => {
  const email = (req as any).userEmail;
  
  const user = await UserDatabase.findByEmail(email);
  
  if (!user) {
    return res.status(404).json({ error: 'User not found' });
  }
  
  res.json({
    email: user.email,
    created_at: user.created_at,
    last_login: user.last_login
  });
});

app.get('/api/auth', async (req, res) => {
  const email = req.signedCookies.user_email;
  let user = null;
  
  if (email) {
    try {
      user = await UserDatabase.findByEmail(email);
    } catch (error) {
      console.error('Error checking user authentication:', error);
    }
  }
  
  res.json({
    isAuthenticated: !!user,
    email: user?.email || null
  });
});

app.post('/api/logout', (req, res) => {
  res.clearCookie('user_email');
  res.json({ message: 'Logged out successfully' });
});


app.get('*', async (req: Request, res: Response) => {
  try {
    const htmlPath = path.join(process.cwd(), 'dist/index.html');
    res.sendFile(htmlPath);
  } catch (error) {
    console.error('Error serving HTML:', error);
    res.status(500).send('Server error');
  }
});

app.listen(PORT, () => {
  console.log(`Frontend server running on http://localhost:${PORT}`);
});