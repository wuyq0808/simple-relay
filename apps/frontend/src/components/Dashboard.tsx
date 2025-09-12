import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import LogoutPanel from './LogoutPanel';
import ApiKeyTable from './ApiKeyTable';
import UsageStats from './UsageStats';
import QRCodeIcon from './QRCodeIcon';
import './Dashboard.scss';

interface DashboardProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

type ActiveTab = 'api-keys' | 'usage';

export default function Dashboard({ userEmail, onMessage }: DashboardProps) {
  const { t } = useTranslation();
  
  // Initialize activeTab from localStorage, fallback to 'api-keys'
  const [activeTab, setActiveTab] = useState<ActiveTab>(() => {
    const savedTab = localStorage.getItem('dashboard-active-tab');
    return (savedTab === 'usage' || savedTab === 'api-keys') ? savedTab : 'api-keys';
  });

  // Save to localStorage whenever activeTab changes
  useEffect(() => {
    localStorage.setItem('dashboard-active-tab', activeTab);
  }, [activeTab]);

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
            <QRCodeIcon />
          </h1>
          <p className="tagline">
            {t('common.tagline')}
          </p>
          
          <nav className="tab-navigation">
            <button 
              className={`tab-button ${activeTab === 'api-keys' ? 'active' : ''}`}
              onClick={() => setActiveTab('api-keys')}
            >
              {t('tabs.apiKeys')}
            </button>
            <button 
              className={`tab-button ${activeTab === 'usage' ? 'active' : ''}`}
              onClick={() => setActiveTab('usage')}
            >
              {t('tabs.usage')}
            </button>
          </nav>
          
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
          {activeTab === 'api-keys' && (
            <div className="api-keys-section">
              <h2>{t('apiKeys.title')}</h2>
              <p className="description">{t('apiKeys.description')}</p>
              
              <ApiKeyTable userEmail={userEmail} onMessage={onMessage} />
            </div>
          )}
          
          {activeTab === 'usage' && (
            <div className="usage-section">
              <h2>{t('usage.title')}</h2>
              <p className="description">{t('usage.description')}</p>
              
              <UsageStats userEmail={userEmail} onMessage={onMessage} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}