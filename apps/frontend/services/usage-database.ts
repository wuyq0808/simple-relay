import { Firestore } from '@google-cloud/firestore';

export interface UsageRecord {
  id: string;
  timestamp: Date;
  model: string;
  input_tokens: number;
  output_tokens: number;
  total_cost: number;
  user_id: string;
}

export interface HourlyUsage {
  Hour: string;
  Model: string;
  InputTokens: number;
  OutputTokens: number;
  TotalCost: number;
  Requests: number;
}

class FirestoreUsageDatabase {
  private db: Firestore;
  private collection = 'hourly_aggregates';

  constructor() {
    const projectId = process.env.GCP_PROJECT_ID;
    const databaseName = process.env.FIRESTORE_DATABASE_NAME;
    
    if (!projectId || !databaseName) {
      throw new Error('GCP_PROJECT_ID and FIRESTORE_DATABASE_NAME environment variables are required');
    }
    
    this.db = new Firestore({
      projectId,
      databaseId: databaseName,
    });
  }

  async findByUserEmail(userEmail: string): Promise<HourlyUsage[]> {
    const collection = this.db.collection(this.collection);
    const snapshot = await collection.where('user_id', '==', userEmail).orderBy('hour', 'desc').get();
    
    if (snapshot.empty) {
      return [];
    }

    const hourlyUsage: HourlyUsage[] = [];
    snapshot.forEach(doc => {
      const data = doc.data();
      const hour = data.hour?.toDate() || new Date(data.hour);
      
      // Format hour as YYYY-MM-DD HH:00
      const hourStr = `${hour.getFullYear()}-${(hour.getMonth() + 1).toString().padStart(2, '0')}-${hour.getDate().toString().padStart(2, '0')} ${hour.getHours().toString().padStart(2, '0')}:00`;
      
      // Process model usage stats from flattened Firestore fields
      const modelUsage = this.extractFlattenedModelUsage(data);
      
      for (const [modelName, stats] of Object.entries(modelUsage)) {
        hourlyUsage.push({
          Hour: hourStr,
          Model: modelName,
          InputTokens: stats.input_tokens || 0,
          OutputTokens: stats.output_tokens || 0,
          TotalCost: stats.total_cost || 0,
          Requests: stats.request_count || 0,
        });
      }
    });

    // Sort by hour desc, then by model asc
    hourlyUsage.sort((a, b) => {
      if (a.Hour !== b.Hour) {
        return b.Hour.localeCompare(a.Hour); // Hour desc
      }
      return a.Model.localeCompare(b.Model); // Model asc
    });

    return hourlyUsage;
  }

  /**
   * Extract flattened model usage fields from Firestore document data.
   * Converts "model_usage.claude-sonnet-4.input_tokens" format
   * to nested object { "claude-sonnet-4": { input_tokens: value } }
   */
  private extractFlattenedModelUsage(data: any): Record<string, any> {
    const modelUsage: Record<string, any> = {};
    
    for (const [key, value] of Object.entries(data)) {
      if (key.startsWith('model_usage.')) {
        const parts = key.split('.');
        if (parts.length === 3) {
          const [, modelName, metric] = parts;
          if (!modelUsage[modelName]) {
            modelUsage[modelName] = {};
          }
          modelUsage[modelName][metric] = value;
        } else {
          console.warn(`Unexpected model_usage field format: ${key}. Expected format: model_usage.{model}.{metric}`);
        }
      }
    }
    
    return modelUsage;
  }
}

// Export singleton instance
export const UsageDatabase = new FirestoreUsageDatabase();