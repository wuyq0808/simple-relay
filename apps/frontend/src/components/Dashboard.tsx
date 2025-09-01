import React from 'react';
import SignedInWidget from './SignedInWidget';

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
    <div className="app-container">
      <div className="app-content">
        <h1 className="app-title">
          AI Fastlane
        </h1>

        <SignedInWidget
          email={userEmail}
          onSignOut={handleSignOut}
        />
      </div>
    </div>
  );
}