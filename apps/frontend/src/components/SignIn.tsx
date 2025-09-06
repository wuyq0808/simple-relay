import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import SignInForm from './SignInForm';

type MessageType = 'success' | 'error' | '';

interface SignInProps {
  onMessage: (message: string, type?: MessageType) => void;
  onSuccess: (email: string) => void;
}

export default function SignIn({ onMessage, onSuccess }: SignInProps) {
  const { t } = useTranslation();
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
        onSuccess(email);
        onMessage(t('auth.verificationCodeSent', 'Verification code sent to your email'), 'success');
      } else {
        onMessage(data.error || t('auth.failedToSendCode', 'Failed to send verification code'), 'error');
      }
    } catch {
      onMessage(t('auth.networkError', 'Network error. Please try again.'), 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <SignInForm
      email={email}
      loading={loading}
      onEmailChange={setEmail}
      onSubmit={handleEmailSubmit}
    />
  );
}