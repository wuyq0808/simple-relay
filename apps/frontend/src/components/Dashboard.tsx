import React, { useState, useEffect } from 'react';
import SignedInWidget from './SignedInWidget';

interface DashboardProps {
  onMessage: (message: string) => void;
}

export default function Dashboard({ onMessage }: DashboardProps) {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch user profile data
    const fetchProfile = async () => {
      try {
        const response = await fetch('/api/profile', {
          credentials: 'include'
        });
        
        if (response.ok) {
          const data = await response.json();
          setEmail(data.email);
        } else {
          // User is not authenticated, reload to get auth flow
          window.location.reload();
        }
      } catch (error) {
        onMessage('Failed to load user profile');
      } finally {
        setLoading(false);
      }
    };

    fetchProfile();
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

  if (loading) {
    return (
      <p className="description">
        Loading...
      </p>
    );
  }

  return (
    <SignedInWidget
      email={email}
      onSignOut={handleSignOut}
    />
  );
}