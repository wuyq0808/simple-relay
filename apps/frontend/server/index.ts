import 'dotenv/config';
import express, { Request, Response } from 'express';
import cors from 'cors';
import path from 'path';
import { fileURLToPath } from 'url';
import { rateLimit } from 'express-rate-limit';
import { sendVerificationEmail } from '../services/email.js';

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
  message: { error: 'Too many requests from this IP. Try again later.' },
  standardHeaders: true,
  legacyHeaders: false,
});

app.use(cors());
app.use(express.json());
// Serve static files from dist directory (same for dev and prod)
app.use(express.static(path.join(process.cwd(), 'dist')));

// In-memory storage for verification codes (use a database in production)  
const verificationCodes = new Map<string, { code: string; timestamp: number }>();

// Generate random 6-digit verification code
function generateVerificationCode(): string {
  return Math.floor(100000 + Math.random() * 900000).toString();
}

// API routes
app.get('/api/health', (_req: Request, res: Response) => {
  res.json({ status: 'ok' });
});

// Signup endpoint - sends verification email
app.post('/api/signup', ipRateLimit, async (req, res) => {
  const { email } = req.body;
  
  if (!email || !email.includes('@')) {
    return res.status(400).json({ error: 'Valid email is required' });
  }

  const verificationCode = generateVerificationCode();
  
  // Store the verification code with timestamp
  verificationCodes.set(email, {
    code: verificationCode,
    timestamp: Date.now()
  });

  try {
    const result = await sendVerificationEmail({
      email,
      verificationCode,
      appName: 'AI Fastlane',
      expirationMinutes: 10
    });

    if (result.success) {
      res.json({ message: 'Verification email sent successfully' });
    } else {
      res.status(500).json({ error: result.error || 'Failed to send verification email' });
    }
  } catch (error) {
    console.error('Email sending error:', error);
    res.status(500).json({ error: 'Failed to send verification email' });
  }
});

// Verify code endpoint
app.post('/api/verify', (req, res) => {
  const { email, code } = req.body;
  
  if (!email || !code) {
    return res.status(400).json({ error: 'Email and verification code are required' });
  }

  const storedData = verificationCodes.get(email);
  
  if (!storedData) {
    return res.status(400).json({ error: 'No verification code found for this email' });
  }

  // Check if code has expired (10 minutes)
  const isExpired = Date.now() - storedData.timestamp > 10 * 60 * 1000;
  
  if (isExpired) {
    verificationCodes.delete(email);
    return res.status(400).json({ error: 'Verification code has expired' });
  }

  if (storedData.code !== code) {
    return res.status(400).json({ error: 'Invalid verification code' });
  }

  // Code is valid - remove it and complete signup
  verificationCodes.delete(email);
  
  // Here you would typically save the user to your database
  console.log(`User verified: ${email}`);
  
  res.json({ message: 'Email verified successfully', email });
});


// Serve React app for all other routes
app.get('*', (_req: Request, res: Response) => {
  res.sendFile(path.join(process.cwd(), 'dist/index.html'));
});

app.listen(PORT, () => {
  console.log(`Frontend server running on http://localhost:${PORT}`);
});