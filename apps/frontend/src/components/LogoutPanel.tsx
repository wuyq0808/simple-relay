import React from 'react';
import './LogoutPanel.scss';

interface LogoutPanelProps {
  email: string;
  onLogout: () => void;
}

export default function LogoutPanel({ email, onLogout }: LogoutPanelProps) {
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
        onClick={onLogout}
        className="primary-button"
      >
        Log Out
      </button>
    </>
  );
}