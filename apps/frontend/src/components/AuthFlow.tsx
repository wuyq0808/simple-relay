import React, { useState } from 'react';
import SignInForm from './SignInForm';
import VerifyCode from './VerifyCode';

type AuthState = 'signin' | 'verify';
type MessageType = 'success' | 'error' | '';

interface AuthFlowProps {
  onMessage: (message: string, type?: MessageType) => void;
  onStateChange: (state: AuthState) => void;
}

export default function AuthFlow({ onMessage, onStateChange }: AuthFlowProps) {
  const [state, setState] = useState<AuthState>('signin');
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    setLoading(true);
    onMessage('');

    try {
      const response = await fetch('/api/signin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });

      const data = await response.json();
      
      if (response.ok) {
        setState('verify');
        onStateChange('verify');
        onMessage('Verification code sent to your email', 'success');
      } else {
        onMessage(data.error || 'Failed to send verification code', 'error');
      }
    } catch {
      onMessage('Network error. Please try again.', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleVerificationSubmit = async (code: string) => {
    if (!code || code.length !== 6) {
      onMessage('Please enter a 6-digit verification code', 'error');
      return;
    }

    setLoading(true);
    onMessage('Verifying...');

    try {
      const response = await fetch('/api/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, code })
      });

      const data = await response.json();
      
      if (response.ok) {
        // Force page reload to get the authenticated HTML
        window.location.reload();
      } else {
        onMessage(data.error || 'Invalid verification code', 'error');
      }
    } catch {
      onMessage('Network error. Please try again.', 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {state === 'signin' && (
        <div key="signin" className="form-container">
          <SignInForm
            email={email}
            loading={loading}
            onEmailChange={setEmail}
            onSubmit={handleEmailSubmit}
          />
        </div>
      )}

      {state === 'verify' && (
        <div key="verify" className="form-container">
          <VerifyCode
            email={email}
            loading={loading}
            onCodeComplete={handleVerificationSubmit}
            onBack={() => {
            setState('signin');
            onStateChange('signin');
          }}
          />
        </div>
      )}
    </>
  );
}