import React, { useState, useEffect } from 'react';
import QRCode from 'qrcode';
import './QRCodeIcon.scss';

export default function QRCodeIcon() {
  const [showQR, setShowQR] = useState(false);
  const [qrCodeDataUrl, setQrCodeDataUrl] = useState<string | null>(null);
  const [qrCodeLink, setQrCodeLink] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const response = await fetch('/api/config');
        const data = await response.json();
        setQrCodeLink(data.qr_code_link);
      } catch (error) {
        console.error('Error fetching config:', error);
      }
    };

    fetchConfig();
  }, []);

  useEffect(() => {
    if (qrCodeLink && !qrCodeDataUrl) {
      setLoading(true);
      QRCode.toDataURL(qrCodeLink, {
        width: 140,
        margin: 1,
        color: {
          dark: '#000000',
          light: '#FFFFFF'
        }
      })
        .then((url) => {
          setQrCodeDataUrl(url);
        })
        .catch((error) => {
          console.error('Error generating QR code:', error);
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [qrCodeLink, qrCodeDataUrl]);

  return (
    <div 
      className="qr-code-container"
      onMouseEnter={() => setShowQR(true)}
      onMouseLeave={() => setShowQR(false)}
    >
      <div className="qr-icon">
        <svg 
          xmlns="http://www.w3.org/2000/svg" 
          width="16" 
          height="16" 
          fill="currentColor" 
          viewBox="0 0 16 16"
        >
          <path d="M2 2h2v2H2z"/>
          <path d="M6 0v6H0V0zM5 1H1v4h4zM4 12H2v2h2z"/>
          <path d="M6 10v6H0v-6zm-5 1v4h4v-4zm11-9h2v2h-2z"/>
          <path d="M10 0v6h6V0zm5 1v4h-4V1zM8 1V0h1v2H8v2H7V1zm0 5V4h1v2zM6 8V7h1V6h1v2h1V7h5v1h-4v1H7V8zm0 0v1H2V8H1v1H0V7h3v1zm10 1h-1V7h1zm-1 0h-1v2h2v-1h-1zm-4 0h2v1h-1v1h-1zm2 3v-1h-1v1h-1v1H9v1h3v-2zm0 0h3v1h-2v1h-1zm-4-1v1h1v-2H7v1z"/>
          <path d="M7 12h1v3h4v1H7zm9 2v2h-3v-1h2v-1z"/>
        </svg>
      </div>
      
      {showQR && (
        <div className="qr-popup">
          {loading && (
            <div className="qr-loading">
              <div className="loading-spinner"></div>
              <p>Generating QR Code...</p>
            </div>
          )}
          
          {qrCodeDataUrl && !loading && (
            <img 
              src={qrCodeDataUrl} 
              alt="QR Code" 
              className="qr-image"
            />
          )}
          
          {!qrCodeLink && !loading && (
            <div className="qr-placeholder">
              <div className="qr-placeholder-content">
                <p className="coming-soon-text">Coming Soon</p>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}