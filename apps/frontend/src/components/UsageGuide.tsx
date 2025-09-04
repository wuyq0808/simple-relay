import { useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation, Trans } from 'react-i18next';
import './UsageGuide.scss';

interface UsageGuideProps {
  isOpen: boolean;
  onClose: () => void;
  backendUrl: string;
  onMessage: (message: string) => void;
}

export default function UsageGuide({ isOpen, onClose, backendUrl, onMessage }: UsageGuideProps) {
  const { t } = useTranslation();
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);

  const copyCommand = async (command: string, label: string) => {
    try {
      await navigator.clipboard.writeText(command);
      setCopiedCommand(label);
      
      // Reset after 1 second
      setTimeout(() => {
        setCopiedCommand(null);
      }, 1000);
    } catch {
      onMessage('Failed to copy command');
    }
  };

  if (!isOpen) return null;

  const installCommand = `npm install -g @anthropic-ai/claude-cli`;
  const oneTimeCommand = `ANTHROPIC_AUTH_TOKEN=your-api-key ANTHROPIC_BASE_URL=${backendUrl} claude`;
  const sessionCommand = `export ANTHROPIC_AUTH_TOKEN=your-api-key\nexport ANTHROPIC_BASE_URL=${backendUrl}\nclaude`;

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content usage-guide" onClick={(e) => e.stopPropagation()}>
        <div className="usage-content">
          <h1>{t('usageGuide.title')}</h1>
          
          <h2>{t('usageGuide.install')}</h2>
          <div className="command-example">
            <div className="code-block-container">
              <pre className="usage-command">
                <code>{installCommand}</code>
              </pre>
              <button 
                className="copy-code-button"
                onClick={() => copyCommand(installCommand, 'install')}
                disabled={copiedCommand === 'install'}
                title="Copy command"
              >
                {copiedCommand === 'install' ? (
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
          
          <h2>{t('usageGuide.envVariables')}</h2>
          <ul>
            <li><strong><code>ANTHROPIC_AUTH_TOKEN</code></strong> - {t('usageGuide.authToken')}</li>
            <li><strong><code>ANTHROPIC_BASE_URL</code></strong> - {t('usageGuide.baseUrl')}</li>
          </ul>

          <h2>{t('usageGuide.shellCommands')}</h2>
          
          <h3>{t('usageGuide.oneTime')}</h3>
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
          
          <h3>{t('usageGuide.sessionExport')}</h3>
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

          <h2>{t('usageGuide.tips')}</h2>
          <ul>
            <li><Trans i18nKey="usageGuide.replaceKey" components={{ code: <code /> }} /></li>
            <li>{t('usageGuide.useButton')}</li>
          </ul>
        </div>
        
        <div className="modal-actions">
          <button className="cancel-button" onClick={onClose}>
            {t('usageGuide.close')}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}