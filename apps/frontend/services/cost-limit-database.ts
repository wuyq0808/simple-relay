import { Firestore } from '@google-cloud/firestore';

export interface DailyCostLimit {
  userId: string;           // Primary key - user email or user ID
  costLimit: number;        // Daily cost limit in dollars
  updateTime: Date;         // When the limit was last updated
}

class FirestoreCostLimitDatabase {
  private db: Firestore;
  private collection = 'daily_cost_limits';

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

  async getCostLimit(userId: string): Promise<DailyCostLimit | null> {
    const docRef = this.db.collection(this.collection).doc(userId);
    const doc = await docRef.get();
    
    if (!doc.exists) {
      return null;
    }
    
    const data = doc.data()!;
    return {
      userId: data.userId,
      costLimit: data.costLimit,
      updateTime: new Date(data.updateTime),
    };
  }

  async setCostLimit(userId: string, costLimit: number): Promise<DailyCostLimit> {
    const newLimit: DailyCostLimit = {
      userId,
      costLimit,
      updateTime: new Date(),
    };
    
    const docRef = this.db.collection(this.collection).doc(userId);
    await docRef.set({
      userId: newLimit.userId,
      costLimit: newLimit.costLimit,
      updateTime: newLimit.updateTime.toISOString(),
    });
    
    return newLimit;
  }

}

// Export singleton instance
export const CostLimitDatabase = new FirestoreCostLimitDatabase();