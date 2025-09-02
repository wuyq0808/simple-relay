import React from 'react';
import './SignedInPanel.scss';

interface SignedInPanelProps {
  email: string;
  onSignOut: () => void;
}

export default function SignedInPanel({ email, onSignOut }: SignedInPanelProps) {
  return (
    <>
      <p className="description">
        Signed in as
      </p>
      <p className="signed-in-email">
        {email}
      </p>

      <hr className="divider" />

      <button
        onClick={onSignOut}
        className="primary-button"
      >
        Sign Out
      </button>
    </>
  );
}