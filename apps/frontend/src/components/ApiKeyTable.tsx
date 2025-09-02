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
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [deleteModal, setDeleteModal] = useState<{ show: boolean; apiKey: string }>({ show: false, apiKey: '' });
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    loadApiKeys();
  }, []);

  const loadApiKeys = async () => {
    try {
      const response = await fetch('/api/api-keys', {
        credentials: 'include'
      });
      if (response.ok) {
        const keys = await response.json();
        setApiKeys(keys);
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
    // Get backend URL from environment variable set by deployment
    return (window as any).__BACKEND_URL__ || process.env.BACKEND_URL || 'https://simple-relay-staging-573916960175.us-central1.run.app';
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      onMessage('API key copied to clipboard');
    } catch (error) {
      onMessage('Failed to copy to clipboard');
    }
  };

  const copyCommand = async (apiKey: string) => {
    const command = `ANTHROPIC_AUTH_TOKEN=${apiKey} ANTHROPIC_BASE_URL=${getBackendUrl()} claude`;
    try {
      await navigator.clipboard.writeText(command);
      onMessage('Command copied to clipboard');
    } catch (error) {
      onMessage('Failed to copy command');
    }
  };

  const maskApiKey = (key: string) => {
    if (key.length <= 8) return key;
    return key.substring(0, 8) + '...' + key.substring(key.length - 4);
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
          disabled={creating || apiKeys.length >= 3}
        >
          {creating ? 'Creating...' : 'Create'}
        </button>
        <span className="key-count">{apiKeys.length}/3 keys</span>
      </div>

      {apiKeys.length === 0 ? (
        <div className="no-keys">
          No API keys yet. Create your first key to get started.
        </div>
      ) : (
        <div className="key-list">
          {apiKeys.map((key) => (
            <div key={key.api_key} className="key-item">
              <div className="key-info">
                <span className="key-value">{maskApiKey(key.api_key)}</span>
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
                  className="copy-button"
                  onClick={() => copyToClipboard(key.api_key)}
                >
                  Copy Key
                </button>
                <button 
                  className="copy-command-button"
                  onClick={() => copyCommand(key.api_key)}
                >
                  Copy Command
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