import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import Loading from './Loading';

interface HourlyUsage {
  Hour: string;
  Model: string;
  InputTokens: number;
  OutputTokens: number;
  TotalCost: number;
  Requests: number;
}

interface UsageStatsProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function UsageStats({ userEmail, onMessage }: UsageStatsProps) {
  const { t } = useTranslation();
  const [usageData, setUsageData] = useState<HourlyUsage[]>([]);
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
      setLoading(false);
      onMessage('Failed to load usage statistics. Please try again.');
      console.error('Usage stats fetch error:', error);
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
                <th>{t('usage.hour')}</th>
                <th>{t('usage.model')}</th>
                <th>{t('usage.requests')}</th>
                <th>{t('usage.input')}</th>
                <th>{t('usage.output')}</th>
                <th>{t('usage.cost')}</th>
              </tr>
            </thead>
            <tbody>
              {usageData.map((usage, index) => (
                <tr key={index}>
                  <td>{usage.Hour}</td>
                  <td>
                    <code className="model-name">{usage.Model}</code>
                  </td>
                  <td>{usage.Requests.toLocaleString()}</td>
                  <td>{usage.InputTokens.toLocaleString()}</td>
                  <td>{usage.OutputTokens.toLocaleString()}</td>
                  <td>${usage.TotalCost.toFixed(6)}</td>
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