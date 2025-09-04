import { useState } from 'react';
import { createPortal } from 'react-dom';
import './UsageGuide.scss';

interface UsageGuideProps {
  isOpen: boolean;
  onClose: () => void;
  backendUrl: string;
  onMessage: (message: string) => void;
}

export default function UsageGuide({ isOpen, onClose, backendUrl, onMessage }: UsageGuideProps) {
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);

  const copyCommand = async (command: string, label: string) => {
    try {
      await navigator.clipboard.writeText(command);
      setCopiedCommand(label);
      onMessage('Command copied to clipboard');
      
      // Reset after 1 second
      setTimeout(() => {
        setCopiedCommand(null);
      }, 1000);
    } catch {
      onMessage('Failed to copy command');
    }
  };

  if (!isOpen) return null;

  const oneTimeCommand = `ANTHROPIC_AUTH_TOKEN=your-api-key ANTHROPIC_BASE_URL=${backendUrl} claude`;
  const sessionCommand = `export ANTHROPIC_AUTH_TOKEN=your-api-key\nexport ANTHROPIC_BASE_URL=${backendUrl}\nclaude`;

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content usage-guide" onClick={(e) => e.stopPropagation()}>
        <div className="usage-content">
          <h1>Usage Guide</h1>
          
          <h2>Environment Variables</h2>
          <ul>
            <li><strong><code>ANTHROPIC_AUTH_TOKEN</code></strong> - Your API key for authentication</li>
            <li><strong><code>ANTHROPIC_BASE_URL</code></strong> - The AI Fastlane API endpoint</li>
          </ul>

          <h2>Shell Commands</h2>
          
          <h3>One-time usage</h3>
          <div className="command-example">
            <div className="code-block-container">
              <pre className="usage-command">
                <code>{oneTimeCommand}</code>
              </pre>
              <button 
                className="copy-code-button"
                onClick={() => copyCommand(oneTimeCommand, 'one-time')}
                disabled={copiedCommand === 'one-time'}
                title="Copy command"
              >
                {copiedCommand === 'one-time' ? (
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M20 6L9 17l-5-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                ) : (
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" stroke="currentColor" strokeWidth="2" fill="none"/>
                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke="currentColor" strokeWidth="2" fill="none"/>
                  </svg>
                )}
              </button>
            </div>
          </div>
          
          <h3>Export for session</h3>
          <div className="command-example">
            <div className="code-block-container">
              <pre className="usage-command">
                <code>{sessionCommand}</code>
              </pre>
              <button 
                className="copy-code-button"
                onClick={() => copyCommand(sessionCommand, 'session')}
                disabled={copiedCommand === 'session'}
                title="Copy command"
              >
                {copiedCommand === 'session' ? (
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M20 6L9 17l-5-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                ) : (
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" stroke="currentColor" strokeWidth="2" fill="none"/>
                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke="currentColor" strokeWidth="2" fill="none"/>
                  </svg>
                )}
              </button>
            </div>
          </div>

          <h2>Tips</h2>
          <ul>
            <li>Replace &apos;your-api-key&apos; with your actual API key</li>
            <li>Use the Copy button next to each key for convenience</li>
          </ul>
        </div>
        
        <div className="modal-actions">
          <button className="cancel-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}