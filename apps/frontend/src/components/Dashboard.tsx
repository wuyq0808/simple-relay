import LogoutPanel from './LogoutPanel';
import ApiKeyTable from './ApiKeyTable';

interface DashboardProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function Dashboard({ userEmail, onMessage }: DashboardProps) {

  const handleSignOut = async () => {
    try {
      await fetch('/api/logout', {
        method: 'POST',
        credentials: 'include'
      });
      // Force page reload to get the unauthenticated HTML
      window.location.reload();
    } catch (error) {
      console.error('Logout error:', error);
      // Still reload even if logout fails
      window.location.reload();
    }
  };

  return (
    <div className="dashboard-container">
      <div className="sidebar">
        <div className="sidebar-content">
          <h1 className="app-title">
            AI Fastlane
          </h1>
          <p className="tagline">
            Never fall behind.
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
            <h2>API Keys</h2>
            <p className="description">Manage your API keys for accessing the service.</p>
            
            <ApiKeyTable userEmail={userEmail} onMessage={onMessage} />
          </div>
        </div>
      </div>
    </div>
  );
}