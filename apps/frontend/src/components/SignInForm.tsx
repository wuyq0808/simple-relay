import React from 'react';
import './SignInForm.scss';

interface SignInFormProps {
  email: string;
  loading: boolean;
  onEmailChange: (email: string) => void;
  onSubmit: (e: React.FormEvent) => void;
}

export default function SignInForm({ email, loading, onEmailChange, onSubmit }: SignInFormProps) {
  return (
    <>
      <p className="description">
        Never fall behind in the AI revolution
      </p>

      <hr className="divider" />

      <form onSubmit={onSubmit}>
        <input
          type="email"
          placeholder="Enter your email"
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
          {loading ? 'Sending...' : 'Continue'}
        </button>
      </form>
    </>
  );
}