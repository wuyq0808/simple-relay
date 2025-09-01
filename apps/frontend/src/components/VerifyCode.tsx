import React from 'react';
import './VerifyCode.scss';

interface VerifyCodeProps {
  email: string;
  verificationCode: string;
  loading: boolean;
  onCodeChange: (code: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  onBack: () => void;
}

export default function VerifyCode({ 
  email, 
  verificationCode, 
  loading, 
  onCodeChange, 
  onSubmit, 
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

      <form onSubmit={onSubmit}>
        <input
          type="text"
          placeholder="6-digit code"
          value={verificationCode}
          onChange={(e) => onCodeChange(e.target.value.replace(/\D/g, '').slice(0, 6))}
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
          onClick={onBack}
          className="secondary-button"
        >
          Back
        </button>
      </form>
    </>
  );
}