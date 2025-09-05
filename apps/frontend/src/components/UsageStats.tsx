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
        <div className="cost-limit-section" style={{ marginBottom: '2rem', padding: '1rem', border: '1px solid #e0e0e0', borderRadius: '8px', backgroundColor: '#f9f9f9' }}>
          <h3 style={{ margin: '0 0 1rem 0', fontSize: '1.2rem' }}>Daily Cost Limit (8pm-8pm UTC)</h3>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '1rem' }}>
            <div>
              <strong>Limit:</strong> ${costLimitInfo.costLimit.toFixed(2)}
            </div>
            <div>
              <strong>Used Today:</strong> ${costLimitInfo.usedToday.toFixed(2)}
            </div>
            <div style={{ color: costLimitInfo.remaining >= 0 ? '#2e7d32' : '#d32f2f' }}>
              <strong>Remaining:</strong> ${costLimitInfo.remaining.toFixed(2)}
            </div>
            <div>
              <strong>Status:</strong> {costLimitInfo.remaining >= 0 ? '✅ Available' : '❌ Limit Exceeded'}
            </div>
          </div>
        </div>
      )}

      {/* Usage Statistics Section */}
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