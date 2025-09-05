import React, { useEffect } from 'react';
import './MessageToast.scss';

interface MessageToastProps {
  message: string;
  onClose: () => void;
  duration?: number;
}

export function MessageToast({ message, onClose, duration = 4000 }: MessageToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose();
    }, duration);

    return () => clearTimeout(timer);
  }, [onClose, duration]);

  return (
    <div className="message-toast">
      {message}
    </div>
  );
}