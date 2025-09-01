import React, { useState, useEffect } from 'react';
import './App.scss';

type AppState = 'signin' | 'verify' | 'signedin' | 'loading';

function App() {
  const [state, setState] = useState<AppState>('loading');
  const [email, setEmail] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  // Check authentication status on page load
  useEffect(() => {
    const checkAuthStatus = async () => {
      // Check if any cookies exist before making the request
      if (!document.cookie) {
        setState('signin');
        return;
      }
      
      try {
        const response = await fetch('/api/profile', {
          credentials: 'include' // Include cookies in request
        });
        
        if (response.ok) {
          const data = await response.json();
          setEmail(data.email);
          setState('signedin');
        } else {
          setState('signin');
        }
      } catch (error) {
        setState('signin');
      }
    };

    checkAuthStatus();
  }, []);

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !email.includes('@')) {
      setMessage('Please enter a valid email address');
      return;
    }

    setLoading(true);
    setMessage('');

    try {
      const response = await fetch('/api/signin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });

      const data = await response.json();
      
      if (response.ok) {
        if (data.user.existing) {
          // User already exists and was signed in automatically
          setState('signedin');
          setMessage('Successfully signed in!');
        } else {
          // New user, need verification
          setState('verify');
          setMessage('Verification code sent to your email');
        }
      } else {
        setMessage(data.error || 'Failed to send verification code');
      }
    } catch (error) {
      setMessage('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleVerificationSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!verificationCode || verificationCode.length !== 6) {
      setMessage('Please enter a 6-digit verification code');
      return;
    }

    setLoading(true);
    setMessage('');

    try {
      const response = await fetch('/api/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, code: verificationCode })
      });

      const data = await response.json();
      
      if (response.ok) {
        setState('signedin');
        setMessage('Successfully signed in!');
      } else {
        setMessage(data.error || 'Invalid verification code');
      }
    } catch (error) {
      setMessage('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleSignOut = async () => {
    try {
      await fetch('/api/logout', {
        method: 'POST',
        credentials: 'include' // Include cookies in request
      });
    } catch (error) {
      console.error('Logout error:', error);
    }
    
    // Clear local state regardless of API call success
    setState('signin');
    setEmail('');
    setVerificationCode('');
    setMessage('');
  };

  return (
    <div className="app-container">
      <div className="app-content">
        
        <h1 className="app-title">
          AI Fastlane
        </h1>

        {state === 'loading' && (
          <p className="description">
            Loading...
          </p>
        )}

        {state === 'signin' && (
          <>
            <p className="description">
              Never fall behind in the AI revolution
            </p>

            <hr className="divider" />

            <form onSubmit={handleEmailSubmit}>
              <input
                type="email"
                placeholder="Enter your email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
                className="form-input"
              />
              
              <button
                type="submit"
                disabled={loading}
                className="primary-button"
              >
                {loading ? 'Sending...' : 'Continue'}
              </button>
            </form>
          </>
        )}

        {state === 'verify' && (
          <>
            <p className="description">
              Enter the verification code sent to:
            </p>
            <p className="email-display">
              {email}
            </p>

            <hr className="divider" />

            <form onSubmit={handleVerificationSubmit}>
              <input
                type="text"
                placeholder="6-digit code"
                value={verificationCode}
                onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                disabled={loading}
                className="verification-input"
                maxLength={6}
              />
              
              <button
                type="submit"
                disabled={loading}
                className="primary-button verify-button"
              >
                {loading ? 'Verifying...' : 'Verify'}
              </button>

              <button
                type="button"
                onClick={() => setState('signin')}
                className="secondary-button"
              >
                Back
              </button>
            </form>
          </>
        )}

        {state === 'signedin' && (
          <>
            <p className="description">
              Signed in as:
            </p>
            <p className="signed-in-email">
              {email}
            </p>

            <hr className="divider" />

            <button
              onClick={handleSignOut}
              className="primary-button"
            >
              Sign Out
            </button>
          </>
        )}

        {message && (
          <>
            <hr className="divider" />
            <p className={`message ${message.includes('error') || message.includes('Invalid') || message.includes('Failed') ? 'error' : ''}`}>
              {message}
            </p>
          </>
        )}

      </div>
    </div>
  );
}

export default App;