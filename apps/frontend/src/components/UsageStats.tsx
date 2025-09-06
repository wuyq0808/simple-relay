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

interface CostLimitInfo {
  costLimit: number;
  usedToday: number;
  remaining: number;
  updateTime: string | null;
  windowStart: string;
  windowEnd: string;
}

interface UsageStatsProps {
  userEmail: string;
  onMessage: (message: string) => void;
}

export default function UsageStats({ userEmail, onMessage }: UsageStatsProps) {
  const { t } = useTranslation();
  const [usageData, setUsageData] = useState<HourlyUsage[]>([]);
  const [costLimitInfo, setCostLimitInfo] = useState<CostLimitInfo | null>(null);
  const [loading, setLoading] = useState(true);

  // Format UTC time window for user's locale
  const getLocalizedTimeWindow = () => {
    const utc8pm = new Date();
    utc8pm.setUTCHours(20, 0, 0, 0);
    
    const localTime = utc8pm.toLocaleTimeString([], { 
      hour: '2-digit', 
      minute: '2-digit',
      timeZoneName: 'short'
    });
    
    return `${t('usage.resetsAt', 'resets at')} ${localTime}`;
  };

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

  const fetchCostLimitInfo = useCallback(async () => {
    try {
      const response = await fetch('/api/cost-limit', {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        setCostLimitInfo(data);
      }
    } catch {
      // Cost limit info is optional, don't show error if it fails
    }
  }, []);

  const fetchUsageStats = useCallback(async () => {
    try {
      setLoading(true);
      
      // Fetch both usage stats and cost limit info in parallel
      const [usageResponse] = await Promise.all([
        fetch('/api/usage-stats', {
          method: 'GET',
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
          },
        }),
        fetchCostLimitInfo()
      ]);

      if (!usageResponse.ok) {
        throw new Error(`HTTP error! status: ${usageResponse.status}`);
      }

      const data = await usageResponse.json();
      setUsageData(data);
      setLoading(false);
    } catch {
      setLoading(false);
      onMessage('Failed to load usage statistics. Please try again.');
    }
  }, [onMessage, fetchCostLimitInfo]);

  useEffect(() => {
    fetchUsageStats();
  }, [userEmail, fetchUsageStats]);

  if (loading) {
    return <Loading />;
  }

  const groupedData = groupUsageByDay(usageData);

  return (
    <div className="usage-stats-container">
      {/* Daily Cost Limit Section */}
      {costLimitInfo && (
        <div className="usage-table-container" style={{ marginBottom: '2rem' }}>
          <table className="usage-table">
            <thead>
              <tr>
                <th colSpan={2}>
                  {t('usage.dailyCostLimit', 'Daily Cost Limit')} - {getLocalizedTimeWindow()}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr className="day-row">
                <td style={{ padding: '12px', verticalAlign: 'middle', width: '100%' }}>
                  {/* Stats */}
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '14px', color: '#333', marginBottom: '8px' }}>
                    <span>
                      {Math.floor(costLimitInfo.costLimit * 1000)} {t('usage.points', 'Points')}
                    </span>
                    <span style={{ color: costLimitInfo.remaining < 0 ? '#dc3545' : (costLimitInfo.remaining / costLimitInfo.costLimit < 0.2) ? '#dc3545' : '#28a745', fontWeight: '600' }}>
                      {costLimitInfo.remaining >= 0 ? `${Math.ceil(costLimitInfo.remaining * 1000)} ${t('usage.remaining', 'remaining')}` : `${Math.ceil(costLimitInfo.remaining * 1000)}`}
                    </span>
                  </div>
                  
                  {/* Progress Bar Container */}
                  <div style={{ 
                    width: '100%', 
                    height: '12px', 
                    backgroundColor: 'white', 
                    overflow: 'hidden',
                    border: '0.5px solid #6c757d'
                  }}>
                    {/* Progress Bar Fill - shows remaining */}
                    <div style={{
                      width: `${Math.max(0, Math.min(100, (costLimitInfo.remaining / costLimitInfo.costLimit) * 100))}%`,
                      height: '100%',
                      backgroundColor: costLimitInfo.remaining < 0 ? '#dc3545' : (costLimitInfo.remaining / costLimitInfo.costLimit < 0.2) ? '#dc3545' : '#28a745',
                      transition: 'width 0.3s ease'
                    }}>
                    </div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      )}

      {/* Usage Statistics Section */}
      {usageData.length > 0 ? (
        <div className="usage-table-container">
          <table className="usage-table">
            <thead>
              <tr>
                <th>{t('usage.date')}</th>
                <th>{t('usage.model')}</th>
                <th>{t('usage.requests')}</th>
                <th>{t('usage.input')}</th>
                <th>{t('usage.output')}</th>
                <th>{t('usage.consumedPoints')}</th>
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
                        {Math.ceil(modelStats.totalCost * 1000)}
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
          <p>{t('usage.noRecords', 'No usage records found')}</p>
        </div>
      )}
    </div>
  );
}