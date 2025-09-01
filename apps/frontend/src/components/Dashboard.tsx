import React, { useState, useEffect } from 'react';
import SignedInWidget from './SignedInWidget';

declare global {
  interface Window {
    __USER_EMAIL__?: string;
  }
}

interface DashboardProps {
  onMessage: (message: string) => void;
}

export default function Dashboard({ onMessage }: DashboardProps) {
  const [email, setEmail] = useState('');

  useEffect(() => {
    // Get email from server-injected data
    const userEmail = window.__USER_EMAIL__;
    
    if (userEmail) {
      setEmail(userEmail);
    } else {
      // No user email found, user shouldn't be on this page
      onMessage('Authentication error - please sign in again');
      setTimeout(() => {
        window.location.reload();
      }, 2000);
    }
  }, [onMessage]);

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
    <SignedInWidget
      email={email}
      onSignOut={handleSignOut}
    />
  );
}