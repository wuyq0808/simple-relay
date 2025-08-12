import { useState, useEffect } from 'react'

function App() {
  const [health, setHealth] = useState<string>('');

  useEffect(() => {
    fetch('/api/health')
      .then(res => res.json())
      .then(data => setHealth(data.status))
      .catch(err => console.error('Error:', err));
  }, []);

  const handleRelay = async () => {
    try {
      const response = await fetch('/api/relay', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ message: 'Hello from frontend!' })
      });
      const data = await response.json();
      alert(JSON.stringify(data, null, 2));
    } catch (err) {
      console.error('Error:', err);
    }
  };

  return (
    <div style={{ fontFamily: 'Inter, system-ui, sans-serif', padding: '2rem', maxWidth: '800px', margin: '0 auto' }}>
      <h1>Simple Relay</h1>
      <p>Backend health: <strong>{health}</strong></p>
      <button onClick={handleRelay} style={{ padding: '0.5rem 1rem', fontSize: '1rem' }}>
        Test Relay Endpoint
      </button>
    </div>
  )
}

export default App