import { useTranslation } from 'react-i18next';
import LogoutPanel from './LogoutPanel';
import ApiKeyTable from './ApiKeyTable';

interface DashboardProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function Dashboard({ userEmail, onMessage }: DashboardProps) {
  const { t } = useTranslation();

  const handleSignOut = async () => {
    try {
      await fetch('/api/logout', {
        method: 'POST',
        credentials: 'include'
      });
      // Force page reload to get the unauthenticated HTML
      window.location.reload();
    } catch {
      // Still reload even if logout fails
      window.location.reload();
    }
  };

  return (
    <div className="dashboard-container">
      <div className="sidebar">
        <div className="sidebar-content">
          <h1 className="app-title">
            {t('app.title')}
          </h1>
          <p className="tagline">
            {t('app.tagline')}
          </p>
          
          <div className="sidebar-bottom">
            <LogoutPanel
              email={userEmail}
              onLogout={handleSignOut}
            />
          </div>
        </div>
      </div>

      <div className="main-panel">
        <div className="main-panel-content">
          <div className="api-keys-section">
            <h2>{t('apiKeys.title')}</h2>
            <p className="description">{t('apiKeys.description')}</p>
            
            <ApiKeyTable userEmail={userEmail} onMessage={onMessage} />
          </div>
        </div>
      </div>
    </div>
  );
}