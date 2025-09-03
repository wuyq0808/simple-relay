import { useState } from 'react';
import '../styles/base.scss';
import SignIn from './SignIn';
import Verify from './Verify';

type MessageType = 'success' | 'error' | '';
type AuthState = 'signin' | 'verify';

function LoginPanel() {
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState<MessageType>('');
  const [authState, setAuthState] = useState<AuthState>('signin');
  const [email, setEmail] = useState('');

  const handleMessage = (msg: string, type: MessageType = '') => {
    setMessage(msg);
    setMessageType(type);
  };

  const handleSignInSuccess = (userEmail: string) => {
    setEmail(userEmail);
    setAuthState('verify');
  };

  const handleBackToSignIn = () => {
    setAuthState('signin');
    setMessage('');
  };

  return (
    <div className="app-container">
      <div key={authState} className="app-content">
        
        <h1 className="app-title">
          AI Fastlane
        </h1>

        {authState === 'signin' && (
          <SignIn 
            onMessage={handleMessage}
            onSuccess={handleSignInSuccess}
          />
        )}

        {authState === 'verify' && (
          <Verify 
            email={email}
            onMessage={handleMessage}
            onBack={handleBackToSignIn}
          />
        )}

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

export default LoginPanel;