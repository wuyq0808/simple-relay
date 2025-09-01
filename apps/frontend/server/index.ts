import 'dotenv/config';
import express, { Request, Response, NextFunction } from 'express';
import cors from 'cors';
import cookieParser from 'cookie-parser';
import path from 'path';
import { fileURLToPath } from 'url';
import rateLimit from 'express-rate-limit';
import { sendVerificationEmail } from '../services/email.js';
import { UserDatabase } from '../services/database.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.PORT || 3000;

// Configure trust proxy for proper IP detection behind proxies
app.set('trust proxy', 1);

// IP rate limiter - 30 requests per hour
const ipRateLimit = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 30, // 30 requests per hour per IP
  message: 'Too many requests from this IP. Try again later.',
});

app.use(cors());
app.use(cookieParser(process.env.COOKIE_SECRET));
app.use(express.json());
// Serve static files from dist directory (same for dev and prod)
app.use(express.static(path.join(process.cwd(), 'dist')));


// Generate random 6-digit verification code
function generateVerificationCode(): string {
  return Math.floor(100000 + Math.random() * 900000).toString();
}

// Set signed login cookie helper
function setLoginCookie(res: Response, email: string): void {
  res.cookie('user_email', email, {
    httpOnly: true,
    secure: process.env.DEPLOYMENT_ENV !== 'development',
    sameSite: 'lax',
    signed: true
    // No maxAge = persistent cookie that never expires
  });
}

// Authentication middleware to verify signed cookies
function requireAuth(req: Request, res: Response, next: NextFunction): void {
  const email = req.signedCookies.user_email;
  
  if (!email) {
    res.status(401).json({ error: 'Authentication required' });
    return;
  }
  
  // Add email to request for use in route handlers
  (req as any).userEmail = email;
  next();
}

// API routes
app.get('/api/health', (_req: Request, res: Response) => {
  res.json({ status: 'ok' });
});

// Unified signup/signin endpoint
app.post('/api/signin', ipRateLimit, async (req, res) => {
  const { email } = req.body;
  
  if (!email || !email.includes('@')) {
    return res.status(400).json({ error: 'Valid email is required' });
  }

  // Check if user already exists
  const existingUser = await UserDatabase.findByEmail(email);
  
  if (existingUser) {
    // User exists - sign them in
    await UserDatabase.updateLastLogin(email);
    
    // Set login cookie
    setLoginCookie(res, email);
    
    return res.json({ 
      message: 'Signed in successfully',
      user: { email, existing: true }
    });
  }

  // User doesn't exist - create new account
  const verificationCode = generateVerificationCode();
  const verificationExpiresAt = new Date(Date.now() + 10 * 60 * 1000); // 10 minutes
  
  // Create user in database
  await UserDatabase.create({
    email,
    last_login: null,
    verification_token: verificationCode,
    verification_expires_at: verificationExpiresAt
  });

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
        user: { email, existing: false }
      });
    } else {
      res.status(500).json({ error: result.error || 'Failed to send verification email' });
    }
  } catch (error) {
    console.error('Email sending error:', error);
    res.status(500).json({ error: 'Failed to send verification email' });
  }
});

// Verify code endpoint
app.post('/api/verify', async (req, res) => {
  const { email, code } = req.body;
  
  if (!email || !code) {
    return res.status(400).json({ error: 'Email and verification code are required' });
  }

  // Find user in database
  const user = await UserDatabase.findByEmail(email);
  
  if (!user) {
    return res.status(400).json({ error: 'No verification code found for this email' });
  }

  // Check if user has a verification token
  if (!user.verification_token) {
    return res.status(400).json({ error: 'No verification code found for this email' });
  }

  // Check if code has expired
  if (!UserDatabase.isVerificationTokenValid(user)) {
    return res.status(400).json({ error: 'Verification code has expired' });
  }

  if (user.verification_token !== code) {
    return res.status(400).json({ error: 'Invalid verification code' });
  }

  // Code is valid - verify the user and update last_login
  await UserDatabase.verifyUser(email);
  
  // Set login cookie after successful verification
  setLoginCookie(res, email);
  
  console.log(`User verified: ${email}`);
  
  res.json({ message: 'Email verified successfully', email });
});

// Profile endpoint - for checking auth status
app.get('/api/profile', requireAuth, async (req, res) => {
  const email = (req as any).userEmail;
  
  // Fetch user data from database
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

// Logout endpoint - clears signed cookie
app.post('/api/logout', (req, res) => {
  res.clearCookie('user_email');
  res.json({ message: 'Logged out successfully' });
});


// Serve React app for all other routes
app.get('*', (_req: Request, res: Response) => {
  res.sendFile(path.join(process.cwd(), 'dist/index.html'));
});

app.listen(PORT, () => {
  console.log(`Frontend server running on http://localhost:${PORT}`);
});