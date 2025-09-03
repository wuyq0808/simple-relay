import { useState, useEffect } from 'react';
import './ApiKeyTable.scss';

interface ApiKey {
  api_key: string;
  user_email: string;
  created_at: string;
}

interface ApiKeyTableProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function ApiKeyTable({ userEmail, onMessage }: ApiKeyTableProps) {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [apiEnabled, setApiEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [deleteModal, setDeleteModal] = useState<{ show: boolean; apiKey: string }>({ show: false, apiKey: '' });
  const [deleting, setDeleting] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);

  useEffect(() => {
    loadApiKeys();
  }, []);

  const loadApiKeys = async () => {
    try {
      const response = await fetch('/api/api-keys', {
        credentials: 'include'
      });
      if (response.ok) {
        const data = await response.json();
        setApiKeys(data.api_keys || data); // Handle both new and old response formats
        setApiEnabled(data.api_enabled !== undefined ? data.api_enabled : true);
      } else {
        onMessage('Failed to load API keys');
      }
    } catch (error) {
      onMessage('Error loading API keys');
    } finally {
      setLoading(false);
    }
  };

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
        onMessage('API key created successfully');
        await loadApiKeys();
      } else {
        const error = await response.json();
        onMessage(`Error: ${error.error}`);
      }
    } catch (error) {
      onMessage('Failed to create API key');
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

  const confirmDelete = async () => {
    if (deleting) return;
    
    setDeleting(true);
    try {
      const response = await fetch(`/api/api-keys/${deleteModal.apiKey}`, {
        method: 'DELETE',
        credentials: 'include'
      });

      if (response.ok) {
        onMessage('API key deleted successfully');
        await loadApiKeys();
      } else {
        onMessage('Failed to delete API key');
      }
    } catch (error) {
      onMessage('Error deleting API key');
    } finally {
      setDeleting(false);
      hideDeleteModal();
    }
  };

  const getBackendUrl = () => {
    const backendUrl = import.meta.env.VITE_BACKEND_URL;
    if (!backendUrl) {
      throw new Error('VITE_BACKEND_URL environment variable is required');
    }
    return backendUrl;
  };

  const copyCommand = async (apiKey: string) => {
    const command = `ANTHROPIC_AUTH_TOKEN=${apiKey} ANTHROPIC_BASE_URL=${getBackendUrl()} claude`;
    try {
      await navigator.clipboard.writeText(command);
      setCopiedCommand(apiKey);
      onMessage('Command copied to clipboard');
      
      // Reset after 1 second
      setTimeout(() => {
        setCopiedCommand(null);
      }, 1000);
    } catch (error) {
      onMessage('Failed to copy command');
    }
  };


  if (loading) {
    return <div className="api-key-loading">Loading API keys...</div>;
  }

  return (
    <div className="api-key-table">
      <div className="api-key-header">
        <button 
          className="create-key-button"
          onClick={createApiKey}
          disabled={creating || apiKeys.length >= 3 || !apiEnabled}
        >
          {creating ? 'Creating...' : 'Create'}
        </button>
        <span className="key-count">{apiKeys.length}/3 keys</span>
      </div>

      {apiKeys.length === 0 ? (
        <div className="no-keys">
          {!apiEnabled ? 'API access is not enabled. Please contact us to enable access.' : 'Create your first key to get started.'}
        </div>
      ) : (
        <div className="key-list">
          {apiKeys.map((key) => (
            <div key={key.api_key} className="key-item">
              <div className="key-info">
                <span className="key-date">
                  Created {new Date(key.created_at).toLocaleDateString('en-CA')}
                </span>
                <div className="key-command">
                  <code>
                    ANTHROPIC_AUTH_TOKEN={key.api_key} ANTHROPIC_BASE_URL={getBackendUrl()} claude
                  </code>
                </div>
              </div>
              <div className="key-actions">
                <button 
                  className="copy-command-button"
                  onClick={() => copyCommand(key.api_key)}
                  disabled={copiedCommand === key.api_key || !apiEnabled}
                >
                  {copiedCommand === key.api_key ? 'Copied' : 'Copy'}
                </button>
                <button 
                  className="delete-button"
                  onClick={() => showDeleteModal(key.api_key)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {deleteModal.show && (
        <div className="modal-overlay" onClick={hideDeleteModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h3>Delete API Key</h3>
            <p>Are you sure you want to delete this API key?</p>
            <p className="api-key-preview">{maskApiKey(deleteModal.apiKey)}</p>
            <div className="modal-actions">
              <button className="cancel-button" onClick={hideDeleteModal} disabled={deleting}>
                Cancel
              </button>
              <button className="confirm-delete-button" onClick={confirmDelete} disabled={deleting}>
                {deleting ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function maskApiKey(apiKey: string): string {
  if (apiKey.length <= 8) return apiKey;
  return apiKey.substring(0, 6) + '...' + apiKey.substring(apiKey.length - 4);
}