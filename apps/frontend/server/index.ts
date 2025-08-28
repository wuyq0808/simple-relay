import 'dotenv/config';
import express from 'express';
import cors from 'cors';
import path from 'path';
import { fileURLToPath } from 'url';
import { sendVerificationEmail } from '../services/email.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.PORT || 3000;

app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname, '../dist')));

// In-memory storage for verification codes (use a database in production)
const verificationCodes = new Map<string, { code: string; timestamp: number }>();

// Generate random 6-digit verification code
function generateVerificationCode(): string {
  return Math.floor(100000 + Math.random() * 900000).toString();
}

// API routes
app.get('/api/health', (req, res) => {
  res.json({ status: 'ok' });
});

// Signup endpoint - sends verification email
app.post('/api/signup', async (req, res) => {
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

app.post('/api/relay', (req, res) => {
  res.json({ message: 'relay endpoint', data: req.body });
});

// Serve React app for all other routes
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, '../dist/index.html'));
});

app.listen(PORT, () => {
  console.log(`Frontend server running on http://localhost:${PORT}`);
});