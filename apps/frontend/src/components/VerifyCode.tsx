import React from 'react';
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
  return (
    <>
      <p className="description">
        Enter the verification code sent to
      </p>
      <p className="email-display">
        {email}
      </p>

      <hr className="divider" />

      <input
        type="text"
        placeholder="6-digit code"
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
        Back
      </button>
    </>
  );
}