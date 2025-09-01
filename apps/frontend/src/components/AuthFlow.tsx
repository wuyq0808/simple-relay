import React, { useState } from 'react';
import SignInForm from './SignInForm';
import VerifyCode from './VerifyCode';

type AuthState = 'signin' | 'verify';

interface AuthFlowProps {
  onMessage: (message: string) => void;
}

export default function AuthFlow({ onMessage }: AuthFlowProps) {
  const [state, setState] = useState<AuthState>('signin');
  const [email, setEmail] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [loading, setLoading] = useState(false);

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !email.includes('@')) {
      onMessage('Please enter a valid email address');
      return;
    }

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
        onMessage('Verification code sent to your email');
      } else {
        onMessage(data.error || 'Failed to send verification code');
      }
    } catch (error) {
      onMessage('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleVerificationSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!verificationCode || verificationCode.length !== 6) {
      onMessage('Please enter a 6-digit verification code');
      return;
    }

    setLoading(true);
    onMessage('');

    try {
      const response = await fetch('/api/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, code: verificationCode })
      });

      const data = await response.json();
      
      if (response.ok) {
        // Force page reload to get the authenticated HTML
        window.location.reload();
      } else {
        onMessage(data.error || 'Invalid verification code');
      }
    } catch (error) {
      onMessage('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {state === 'signin' && (
        <SignInForm
          email={email}
          loading={loading}
          onEmailChange={setEmail}
          onSubmit={handleEmailSubmit}
        />
      )}

      {state === 'verify' && (
        <VerifyCode
          email={email}
          verificationCode={verificationCode}
          loading={loading}
          onCodeChange={setVerificationCode}
          onSubmit={handleVerificationSubmit}
          onBack={() => setState('signin')}
        />
      )}
    </>
  );
}