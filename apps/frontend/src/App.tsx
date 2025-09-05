import React, { useState, useEffect } from 'react';
import './styles/base.scss';
import LoginPanel from './components/LoginPanel';
import Dashboard from './components/Dashboard';
import Loading from './components/Loading';
import { MessageToast } from './components/MessageToast';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [userEmail, setUserEmail] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showLoadingText, setShowLoadingText] = useState(false);
  const [message, setMessage] = useState('');

  // Auto-dismiss message after 3 seconds
  useEffect(() => {
    if (message) {
      const timer = setTimeout(() => {
        setMessage('');
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [message]);

  useEffect(() => {
    const loadingTimer = setTimeout(() => {
      setShowLoadingText(true);
    }, 3000);

    fetch('/api/auth', { credentials: 'include' })
      .then(res => res.json())
      .then(data => {
        setIsAuthenticated(data.isAuthenticated);
        setUserEmail(data.email);
      })
      .catch(() => {
        setIsAuthenticated(false);
        setUserEmail(null);
      })
      .finally(() => {
        clearTimeout(loadingTimer);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <div className="app-container loading">
        <div className="app-content loading">
          {showLoadingText && <Loading />}
        </div>
      </div>
    );
  }

  if (isAuthenticated) {
    return (
      <>
        <Dashboard userEmail={userEmail!} onMessage={setMessage} />
        {message && (
          <MessageToast message={message} />
        )}
      </>
    );
  }

  return <LoginPanel />;
}

export default App;