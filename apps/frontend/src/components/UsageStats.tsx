import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import Loading from './Loading';

interface DailyUsage {
  Date: string;
  Model: string;
  InputTokens: number;
  OutputTokens: number;
}

interface UsageStatsProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function UsageStats({ userEmail, onMessage }: UsageStatsProps) {
  const { t } = useTranslation();
  const [usageData, setUsageData] = useState<DailyUsage[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchUsageStats();
  }, [userEmail]);

  const fetchUsageStats = async () => {
    try {
      setLoading(true);
      
      const response = await fetch('/api/usage-stats', {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setUsageData(data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching usage stats:', error);
      // Keep loading state on error, don't show error message
      // setLoading stays true to show loading state
    }
  };

  if (loading) {
    return <Loading />;
  }

  return (
    <div className="usage-stats-container">
      {usageData.length > 0 ? (
        <div className="usage-table-container">
          <table className="usage-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Model</th>
                <th>Input</th>
                <th>Output</th>
              </tr>
            </thead>
            <tbody>
              {usageData.map((usage, index) => (
                <tr key={index}>
                  <td>{usage.Date}</td>
                  <td>
                    <code className="model-name">{usage.Model}</code>
                  </td>
                  <td>{usage.InputTokens.toLocaleString()}</td>
                  <td>{usage.OutputTokens.toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="empty-state">
          <p>No usage records found</p>
        </div>
      )}
    </div>
  );
}