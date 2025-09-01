import React from 'react';
import './SignedInWidget.scss';

interface SignedInWidgetProps {
  email: string;
  onSignOut: () => void;
}

export default function SignedInWidget({ email, onSignOut }: SignedInWidgetProps) {
  return (
    <>
      <p className="description">
        Signed in as:
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