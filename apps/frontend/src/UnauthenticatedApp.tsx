import React, { useState } from 'react';
import './App.scss';
import AuthFlow from './components/AuthFlow';

function UnauthenticatedApp() {
  const [message, setMessage] = useState('');

  return (
    <div className="app-container">
      <div className="app-content">
        
        <h1 className="app-title">
          AI Fastlane
        </h1>

        <AuthFlow onMessage={setMessage} />

        {message && (
          <>
            <hr className="divider" />
            <p className={`message ${message.includes('error') || message.includes('Invalid') || message.includes('Failed') ? 'error' : ''}`}>
              {message}
            </p>
          </>
        )}

      </div>
    </div>
  );
}

export default UnauthenticatedApp;