import React, { useState } from 'react';
import './styles/base.scss';
import AuthFlow from './components/AuthFlow';

type MessageType = 'success' | 'error' | '';

function UnauthenticatedApp() {
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState<MessageType>('');

  const handleMessage = (msg: string, type: MessageType = '') => {
    setMessage(msg);
    setMessageType(type);
  };

  return (
    <div className="app-container">
      <div className="app-content">
        
        <h1 className="app-title">
          AI Fastlane
        </h1>

        <AuthFlow onMessage={handleMessage} />

        {message && (
          <>
            <hr className="divider" />
            <p className={`message ${messageType}`}>
              {message}
            </p>
          </>
        )}

      </div>
    </div>
  );
}

export default UnauthenticatedApp;