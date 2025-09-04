import React from 'react';
import { useTranslation } from 'react-i18next';
import './LogoutPanel.scss';

interface LogoutPanelProps {
  email: string;
  onLogout: () => void;
}

export default function LogoutPanel({ email, onLogout }: LogoutPanelProps) {
  const { t } = useTranslation();
  
  return (
    <>
      <p className="description">
        {t('auth.signedInAs')}
      </p>
      <p className="signed-in-email">
        {email}
      </p>

      <hr className="divider" />

      <button
        onClick={onLogout}
        className="primary-button"
      >
        {t('auth.signOut')}
      </button>
    </>
  );
}