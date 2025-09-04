import React, { useState } from 'react';
import VerifyCode from './VerifyCode';

type MessageType = 'success' | 'error' | '';

interface VerifyProps {
  email: string;
  onMessage: (message: string, type?: MessageType) => void;
  onBack: () => void;
}

export default function Verify({ email, onMessage, onBack }: VerifyProps) {
  const [loading, setLoading] = useState(false);

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
    <VerifyCode
      email={email}
      loading={loading}
      onCodeComplete={handleVerificationSubmit}
      onBack={onBack}
    />
  );
}