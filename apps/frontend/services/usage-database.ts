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

export interface DailyUsage {
  Date: string;
  Model: string;
  InputTokens: number;
  OutputTokens: number;
}

class FirestoreUsageDatabase {
  private db: Firestore;
  private collection = 'usage_records';

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

  async findByUserEmail(userEmail: string): Promise<DailyUsage[]> {
    const collection = this.db.collection(this.collection);
    const snapshot = await collection.where('user_id', '==', userEmail).get();
    
    if (snapshot.empty) {
      return [];
    }

    const records: UsageRecord[] = [];
    snapshot.forEach(doc => {
      const data = doc.data();
      records.push({
        id: data.id || doc.id,
        timestamp: data.timestamp?.toDate() || new Date(data.timestamp),
        model: data.model || 'unknown-model',
        input_tokens: parseInt(data.input_tokens) || 0,
        output_tokens: parseInt(data.output_tokens) || 0,
        total_cost: parseFloat(data.total_cost) || 0,
        user_id: data.user_id,
      });
    });

    // Group by date and model
    const dailyUsage = new Map<string, DailyUsage>();
    
    for (const record of records) {
      const date = record.timestamp.toLocaleDateString();
      const key = `${date}-${record.model}`;
      
      if (dailyUsage.has(key)) {
        const existing = dailyUsage.get(key)!;
        existing.InputTokens += record.input_tokens;
        existing.OutputTokens += record.output_tokens;
      } else {
        dailyUsage.set(key, {
          Date: date,
          Model: record.model,
          InputTokens: record.input_tokens,
          OutputTokens: record.output_tokens,
        });
      }
    }

    // Convert to array and sort by date desc, then by model asc
    const usage = Array.from(dailyUsage.values());
    usage.sort((a, b) => {
      if (a.Date !== b.Date) {
        return b.Date.localeCompare(a.Date); // Date desc
      }
      return a.Model.localeCompare(b.Model); // Model asc
    });

    return usage;
  }
}

// Export singleton instance
export const UsageDatabase = new FirestoreUsageDatabase();