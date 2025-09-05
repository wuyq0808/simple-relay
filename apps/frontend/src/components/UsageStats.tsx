import { useState, useEffect, useCallback } from 'react';
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

interface GroupedUsage {
  day: string;
  models: Record<string, {
    requests: number;
    inputTokens: number;
    outputTokens: number;
    totalCost: number;
  }>;
  totalRequests: number;
  totalInputTokens: number;
  totalOutputTokens: number;
  totalCost: number;
}

interface UsageStatsProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function UsageStats({ userEmail, onMessage }: UsageStatsProps) {
  const { t } = useTranslation();
  const [usageData, setUsageData] = useState<HourlyUsage[]>([]);
  const [loading, setLoading] = useState(true);

  const groupUsageByDay = (data: HourlyUsage[]): GroupedUsage[] => {
    const groups: Record<string, GroupedUsage> = {};
    
    data.forEach(usage => {
      // Extract date from hour string (YYYY-MM-DD HH:mm)
      const day = usage.Hour.split(' ')[0];
      
      if (!groups[day]) {
        groups[day] = {
          day: day,
          models: {},
          totalRequests: 0,
          totalInputTokens: 0,
          totalOutputTokens: 0,
          totalCost: 0,
        };
      }
      
      // Aggregate by model within the day
      if (!groups[day].models[usage.Model]) {
        groups[day].models[usage.Model] = {
          requests: 0,
          inputTokens: 0,
          outputTokens: 0,
          totalCost: 0,
        };
      }
      
      groups[day].models[usage.Model].requests += usage.Requests;
      groups[day].models[usage.Model].inputTokens += usage.InputTokens;
      groups[day].models[usage.Model].outputTokens += usage.OutputTokens;
      groups[day].models[usage.Model].totalCost += usage.TotalCost;
      
      groups[day].totalRequests += usage.Requests;
      groups[day].totalInputTokens += usage.InputTokens;
      groups[day].totalOutputTokens += usage.OutputTokens;
      groups[day].totalCost += usage.TotalCost;
    });
    
    return Object.values(groups).sort((a, b) => b.day.localeCompare(a.day));
  };

  const fetchUsageStats = useCallback(async () => {
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
      // eslint-disable-next-line no-console
      console.error('Usage stats fetch error:', error);
    }
  }, [onMessage]);

  useEffect(() => {
    fetchUsageStats();
  }, [userEmail, fetchUsageStats]);

  if (loading) {
    return <Loading />;
  }

  const groupedData = groupUsageByDay(usageData);

  return (
    <div className="usage-stats-container">
      {usageData.length > 0 ? (
        <div className="usage-table-container">
          <table className="usage-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>{t('usage.model')}</th>
                <th>{t('usage.requests')}</th>
                <th>{t('usage.input')}</th>
                <th>{t('usage.output')}</th>
                <th>{t('usage.points')}</th>
              </tr>
            </thead>
            <tbody>
              {groupedData.map((group) => (
                <tr key={group.day} className="day-row">
                  <td className="day-cell">{group.day}</td>
                  <td className="models-cell">
                    {Object.entries(group.models).map(([modelName, _modelStats], index) => (
                      <span key={modelName}>
                        <code className="model-name">{modelName}</code>
                        {index < Object.entries(group.models).length - 1 && <br />}
                      </span>
                    ))}
                  </td>
                  <td className="stats-cell">
                    {Object.entries(group.models).map(([modelName, modelStats], index) => (
                      <span key={modelName}>
                        {modelStats.requests.toLocaleString()}
                        {index < Object.entries(group.models).length - 1 && <br />}
                      </span>
                    ))}
                  </td>
                  <td className="stats-cell">
                    {Object.entries(group.models).map(([modelName, modelStats], index) => (
                      <span key={modelName}>
                        {modelStats.inputTokens.toLocaleString()}
                        {index < Object.entries(group.models).length - 1 && <br />}
                      </span>
                    ))}
                  </td>
                  <td className="stats-cell">
                    {Object.entries(group.models).map(([modelName, modelStats], index) => (
                      <span key={modelName}>
                        {modelStats.outputTokens.toLocaleString()}
                        {index < Object.entries(group.models).length - 1 && <br />}
                      </span>
                    ))}
                  </td>
                  <td className="stats-cell">
                    {Object.entries(group.models).map(([modelName, modelStats], index) => (
                      <span key={modelName}>
                        {Math.floor(modelStats.totalCost * 100)}
                        {index < Object.entries(group.models).length - 1 && <br />}
                      </span>
                    ))}
                  </td>
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