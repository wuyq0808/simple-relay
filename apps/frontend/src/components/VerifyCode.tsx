import React from 'react';
import { useTranslation } from 'react-i18next';
import './VerifyCode.scss';

interface VerifyCodeProps {
  email: string;
  loading: boolean;
  onCodeComplete: (code: string) => void;
  onBack: () => void;
}

export default function VerifyCode({ 
  email, 
  loading, 
  onCodeComplete, 
  onBack 
}: VerifyCodeProps) {
  const { t } = useTranslation();
  
  return (
    <>
      <p className="description">
        {t('auth.verificationSent', 'Enter the verification code sent to')}
      </p>
      <p className="email-display">
        {email}
      </p>

      <hr className="divider" />

      <input
        type="text"
        placeholder={t('auth.codePlaceholder', '6-digit code')}
        onChange={(e) => {
          const code = e.target.value.replace(/\D/g, '').slice(0, 6);
          e.target.value = code;
          if (code.length === 6 && !loading) {
            onCodeComplete(code);
          }
        }}
        disabled={loading}
        className="verification-input"
        maxLength={6}
      />
      
      
      <button
        type="button"
        onClick={onBack}
        className="secondary-button"
      >
        {t('auth.back', 'Back')}
      </button>
    </>
  );
}