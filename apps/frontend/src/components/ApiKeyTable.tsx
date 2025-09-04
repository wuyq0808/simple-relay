import { useState, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import './ApiKeyTable.scss';
import Loading from './Loading';
import UsageGuide from './UsageGuide';

interface ApiKey {
  api_key: string;
  user_email: string;
  created_at: string;
}

interface ApiKeyTableProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function ApiKeyTable({ userEmail: _userEmail, onMessage }: ApiKeyTableProps) {
  const { t } = useTranslation();
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [apiEnabled, setApiEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [deleteModal, setDeleteModal] = useState<{ show: boolean; apiKey: string }>({ show: false, apiKey: '' });
  const [deleting, setDeleting] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);
  const [usageGuideModal, setUsageGuideModal] = useState(false);
  const [accessApprovalPending, setAccessApprovalPending] = useState(false);

  const loadApiKeys = useCallback(async () => {
    try {
      const response = await fetch('/api/api-keys', {
        credentials: 'include'
      });
      if (response.ok) {
        const data = await response.json();
        setApiKeys(data.api_keys || data); // Handle both new and old response formats
        setApiEnabled(data.api_enabled !== undefined ? data.api_enabled : true);
        setAccessApprovalPending(data.access_approval_pending || false);
      } else {
        onMessage(t('apiKeys.messages.loadError'));
      }
    } catch {
      onMessage(t('apiKeys.messages.loadError'));
    } finally {
      setLoading(false);
    }
  }, [onMessage, t]);

  useEffect(() => {
    loadApiKeys();
  }, [loadApiKeys]);

  const createApiKey = async () => {
    if (creating) return;
    
    setCreating(true);
    try {
      const response = await fetch('/api/api-keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (response.ok) {
        onMessage(t('apiKeys.messages.created'));
        await loadApiKeys();
      } else {
        const error = await response.json();
        onMessage(`Error: ${error.error}`);
      }
    } catch {
      onMessage(t('apiKeys.messages.createError'));
    } finally {
      setCreating(false);
    }
  };

  const showDeleteModal = (apiKey: string) => {
    setDeleteModal({ show: true, apiKey });
  };

  const hideDeleteModal = () => {
    setDeleteModal({ show: false, apiKey: '' });
  };

  const showUsageGuide = () => {
    setUsageGuideModal(true);
  };

  const hideUsageGuide = () => {
    setUsageGuideModal(false);
  };

  const confirmDelete = async () => {
    if (deleting) return;
    
    setDeleting(true);
    try {
      const response = await fetch(`/api/api-keys/${deleteModal.apiKey}`, {
        method: 'DELETE',
        credentials: 'include'
      });

      if (response.ok) {
        onMessage(t('apiKeys.messages.deleted'));
        await loadApiKeys();
      } else {
        onMessage(t('apiKeys.messages.deleteError'));
      }
    } catch {
      onMessage(t('apiKeys.messages.deleteError'));
    } finally {
      setDeleting(false);
      hideDeleteModal();
    }
  };

  const getBackendUrl = () => {
    const backendUrl = (import.meta as unknown as { env: { VITE_BACKEND_URL?: string } }).env.VITE_BACKEND_URL;
    if (!backendUrl) {
      throw new Error('VITE_BACKEND_URL environment variable is required');
    }
    return backendUrl;
  };

  const maskApiKey = (apiKey: string) => {
    const firstPart = apiKey.slice(0, 7);
    const lastPart = apiKey.slice(-4);
    const middleLength = apiKey.length - 11; // total length - first 7 - last 4
    return firstPart + '*'.repeat(middleLength) + lastPart;
  };

  const copyCommand = async (apiKey: string) => {
    try {
      await navigator.clipboard.writeText(apiKey);
      setCopiedCommand(apiKey);
      
      // Reset after 1 second
      setTimeout(() => {
        setCopiedCommand(null);
      }, 1000);
    } catch {
      onMessage(t('apiKeys.messages.copyError'));
    }
  };

  const requestAccess = async () => {
    try {
      const response = await fetch('/api/request-access', {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json'
        }
      });

      if (response.ok) {
        setAccessApprovalPending(true);
        onMessage(t('apiKeys.messages.accessRequested'));
      } else {
        onMessage(t('apiKeys.messages.error'));
      }
    } catch {
      onMessage(t('apiKeys.messages.error'));
    }
  };

  if (loading) {
    return <Loading />;
  }

  return (
    <div className="api-key-table">
      <div className="api-key-header">
        <div className="header-left">
          <button 
            className="create-key-button"
            onClick={createApiKey}
            disabled={creating || apiKeys.length >= 3 || !apiEnabled}
          >
            {creating ? t('apiKeys.creating') : t('apiKeys.create')}
          </button>
          <button 
            className="usage-guide-button"
            onClick={showUsageGuide}
          >
            {t('apiKeys.usageGuide')}
          </button>
        </div>
      </div>

      {apiKeys.length === 0 ? (
        <div className="no-keys">
          {!apiEnabled ? (
            accessApprovalPending ? (
              t('apiKeys.accessRequested')
            ) : (
              <button 
                className="request-access-link"
                onClick={requestAccess}
              >
                {t('apiKeys.requestAccess')}
              </button>
            )
          ) : (
            t('apiKeys.noKeys')
          )}
        </div>
      ) : (
        <div className="key-list">
          {apiKeys.map((key) => (
            <div key={key.api_key} className="key-item">
              <div className="key-info">
                <span className="key-date">
                  {t('apiKeys.created', { date: new Date(key.created_at).toLocaleDateString() })}
                </span>
                <div className="key-command-row">
                  <div className="key-display">
                    <code>{maskApiKey(key.api_key)}</code>
                  </div>
                  <div className="key-buttons">
                    <button 
                      className="copy-key-button"
                      onClick={() => copyCommand(key.api_key)}
                      disabled={copiedCommand === key.api_key || !apiEnabled}
                      title={t('apiKeys.copy')}
                    >
                      {copiedCommand === key.api_key ? (
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                          <path d="M20 6L9 17l-5-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                        </svg>
                      ) : (
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" stroke="currentColor" strokeWidth="2" fill="none"/>
                          <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke="currentColor" strokeWidth="2" fill="none"/>
                        </svg>
                      )}
                    </button>
                    <button 
                      className="delete-button"
                      onClick={() => showDeleteModal(key.api_key)}
                      title={t('apiKeys.delete')}
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14zM10 11v6M14 11v6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {deleteModal.show && createPortal(
        <div className="modal-overlay" onClick={hideDeleteModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h3>{t('apiKeys.deleteTitle')}</h3>
            <p>{t('apiKeys.deleteMessage')}</p>
            <p className="api-key-preview">{maskApiKey(deleteModal.apiKey)}</p>
            <div className="modal-actions">
              <button className="cancel-button" onClick={hideDeleteModal} disabled={deleting}>
                {t('apiKeys.cancel')}
              </button>
              <button className="confirm-delete-button" onClick={confirmDelete} disabled={deleting}>
                {deleting ? t('apiKeys.deleting') : t('apiKeys.delete')}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}

      <UsageGuide 
        isOpen={usageGuideModal} 
        onClose={hideUsageGuide} 
        backendUrl={getBackendUrl()} 
        onMessage={onMessage}
      />
    </div>
  );
}

