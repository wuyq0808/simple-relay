import 'dotenv/config';
import express, { Request, Response, NextFunction } from 'express';
import cors from 'cors';
import cookieParser from 'cookie-parser';
import path from 'path';
import rateLimit from 'express-rate-limit';
import baseX from 'base-x';
import { sendVerificationEmail } from '../services/email.js';

const BASE62 = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
const base62 = baseX(BASE62);
import { UserDatabase } from '../services/user-database.js';
import { ConfigService } from '../services/config.js';
import { ApiKeyDatabase } from '../services/api-key-database.js';
import { UsageDatabase, HourlyUsage } from '../services/usage-database.js';
import { PointsLimitDatabase } from '../services/points-limit-database.js';
import { 
  validateSignIn,
  validateEmailVerification
} from './middleware/validation.js';


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
  
  if (!email || typeof email !== 'string') {
    res.status(401).json({ error: 'Authentication required' });
    return;
  }
  
  next();
}


app.get('/api/health', (_req: Request, res: Response) => {
  res.json({ status: 'ok' });
});

app.post('/api/signin', ipRateLimit, validateSignIn, async (req, res) => {
  // req.body is now validated and properly typed by express-zod-safe
  const { email } = req.body;

  const existingUser = await UserDatabase.findByEmail(email);
  
  // Check if signup is enabled for new users
  if (!existingUser) {
    try {
      const signupEnabled = await ConfigService.getConfig('signup_enabled');
      if (signupEnabled === false) {
        return res.status(403).json({ error: 'Sign up is currently disabled' });
      }

      // Check if max user limit is reached
      const maxUsers = await ConfigService.getConfig('max_registered_users');
      if (typeof maxUsers !== 'number' || maxUsers <= 0) {
        console.error('max_registered_users config must be a positive number, got:', maxUsers);
        return res.status(500).json({ error: 'Configuration service error' });
      }
      
      const currentUserCount = await UserDatabase.countUsers();
      if (currentUserCount >= maxUsers) {
        return res.status(403).json({ error: 'Maximum number of registered users reached' });
      }
    } catch (error) {
      console.error('Error checking signup config:', error);
      return res.status(500).json({ error: 'Configuration service unavailable' });
    }
  }

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

app.post('/api/verify', validateEmailVerification, async (req, res) => {
  // req.body is now validated and properly typed by express-zod-safe
  const { email, code } = req.body;

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
  const email = req.signedCookies.user_email;
  
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

app.post('/api/logout', (_req, res) => {
  res.clearCookie('user_email');
  res.json({ message: 'Logged out successfully' });
});

// API Key management endpoints
app.get('/api/api-keys', requireAuth, async (req, res) => {
  try {
    const email = req.signedCookies.user_email;
    
    // Get user to check api_enabled status
    const user = await UserDatabase.findByEmail(email);
    if (!user) {
      return res.status(404).json({ error: 'User not found' });
    }
    
    // Get API keys from separate collection
    const apiKeys = await ApiKeyDatabase.findByUserEmail(email);
    
    res.json({
      api_keys: apiKeys,
      api_enabled: user.api_enabled,
      access_approval_pending: user.access_approval_pending || false
    });
  } catch (error) {
    console.error('Error fetching API keys:', error);
    res.status(500).json({ error: 'Failed to fetch API keys' });
  }
});

app.post('/api/api-keys', requireAuth, async (req, res) => {
  try {
    const email = req.signedCookies.user_email;
    
    // Check if API is enabled for this user
    const user = await UserDatabase.findByEmail(email);
    if (!user) {
      return res.status(404).json({ error: 'User not found' });
    }
    
    if (!user.api_enabled) {
      return res.status(403).json({ error: 'API access is not enabled for this user' });
    }
    
    // Generate API key on server (base62: 0-9, A-Z, a-z)
    const randomBytes = crypto.getRandomValues(new Uint8Array(20));
    const apiKey = 'sk-afl-' + base62.encode(randomBytes);
    
    const newBinding = await ApiKeyDatabase.create({
      api_key: apiKey,
      user_email: email,
    });
    
    res.json(newBinding);
  } catch (error) {
    console.error('Error creating API key:', error);
    if (error instanceof Error && error.message.includes('maximum of 3 API keys')) {
      return res.status(400).json({ error: error.message });
    }
    res.status(500).json({ error: 'Failed to create API key' });
  }
});

app.delete('/api/api-keys/:key', requireAuth, async (req, res) => {
  try {
    const email = req.signedCookies.user_email;
    const apiKey = req.params.key;
    
    // Verify the API key belongs to the user
    const binding = await ApiKeyDatabase.findByApiKey(apiKey);
    if (!binding || binding.user_email !== email) {
      return res.status(404).json({ error: 'API key not found' });
    }
    
    await ApiKeyDatabase.deleteApiKey(apiKey);
    res.json({ message: 'API key deleted successfully' });
  } catch (error) {
    console.error('Error deleting API key:', error);
    res.status(500).json({ error: 'Failed to delete API key' });
  }
});

app.post('/api/request-access', requireAuth, async (req, res) => {
  try {
    const email = req.signedCookies.user_email;
    
    // Update user to mark access approval as pending
    await UserDatabase.updateUser(email, {
      access_approval_pending: true
    });
    
    res.json({ message: 'Access request submitted successfully' });
  } catch (error) {
    console.error('Error requesting access:', error);
    res.status(500).json({ error: 'Failed to submit access request' });
  }
});

app.get('/api/usage-stats', requireAuth, async (req, res) => {
  try {
    const email = req.signedCookies.user_email;
    
    // Query actual usage data from Firestore
    const usageData = await UsageDatabase.findByUserEmail(email);
    
    res.json(usageData);
  } catch (error) {
    console.error('Error fetching usage stats:', error);
    res.status(500).json({ error: 'Failed to fetch usage stats' });
  }
});

app.get('/api/points-limit', requireAuth, async (req, res) => {
  try {
    const email = req.signedCookies.user_email;
    
    // Get daily points limit for user
    const pointsLimit = await PointsLimitDatabase.getPointsLimit(email);
    
    // Calculate today's usage (8pm-8pm UTC window)
    const now = new Date();
    let windowStart: Date;
    if (now.getUTCHours() >= 20) {
      // If it's after 8pm UTC today, the window started at 8pm UTC today
      windowStart = new Date(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), 20, 0, 0, 0);
    } else {
      // If it's before 8pm UTC today, the window started at 8pm UTC yesterday
      const yesterday = new Date(now);
      yesterday.setUTCDate(now.getUTCDate() - 1);
      windowStart = new Date(yesterday.getUTCFullYear(), yesterday.getUTCMonth(), yesterday.getUTCDate(), 20, 0, 0, 0);
    }
    
    const windowEnd = new Date(windowStart);
    windowEnd.setUTCHours(windowStart.getUTCHours() + 24);
    
    // Get usage data for current window
    const todayUsage = await UsageDatabase.findByUserEmailAndTimeRange(email, windowStart, windowEnd);
    const usedToday = todayUsage.reduce((sum: number, usage: HourlyUsage) => sum + usage.TotalPoints, 0);
    
    const dailyLimit = pointsLimit?.pointsLimit || 0;
    const remaining = dailyLimit - usedToday;
    
    res.json({
      pointsLimit: dailyLimit,
      usedToday: usedToday,
      remaining: remaining,
      updateTime: pointsLimit?.updateTime || null,
      windowStart: windowStart.toISOString(),
      windowEnd: windowEnd.toISOString()
    });
  } catch (error) {
    console.error('Error fetching points limit:', error);
    res.status(500).json({ error: 'Failed to fetch points limit' });
  }
});

app.get('*', (_req: Request, res: Response) => {
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