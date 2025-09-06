import React from 'react';
import { useTranslation } from 'react-i18next';
import './SignInForm.scss';

interface SignInFormProps {
  email: string;
  loading: boolean;
  onEmailChange: (email: string) => void;
  onSubmit: (e: React.FormEvent) => void;
}

export default function SignInForm({ email, loading, onEmailChange, onSubmit }: SignInFormProps) {
  const { t } = useTranslation();
  
  return (
    <>
      <p className="description">
        {t('auth.tagline', 'Never fall behind in the AI revolution')}
      </p>

      <hr className="divider" />

      <form onSubmit={onSubmit}>
        <input
          type="email"
          placeholder={t('auth.emailPlaceholder', 'Enter your email')}
          value={email}
          onChange={(e) => onEmailChange(e.target.value)}
          disabled={loading}
          className="form-input"
        />
        
        <button
          type="submit"
          disabled={loading}
          className="primary-button"
        >
          {loading ? t('auth.sending', 'Sending...') : t('auth.continue', 'Continue')}
        </button>
      </form>
    </>
  );
}