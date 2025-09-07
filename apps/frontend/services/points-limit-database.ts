import { Firestore } from '@google-cloud/firestore';

export interface DailyPointsLimit {
  userId: string;           // Primary key - user email or user ID
  pointsLimit: number;      // Daily points limit 
  updateTime: Date;         // When the limit was last updated
}

class FirestorePointsLimitDatabase {
  private db: Firestore;
  private collection = 'daily_points_limits';

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

  async getPointsLimit(userId: string): Promise<DailyPointsLimit | null> {
    const docRef = this.db.collection(this.collection).doc(userId);
    const doc = await docRef.get();
    
    if (!doc.exists) {
      return null;
    }
    
    const data = doc.data()!;
    return {
      userId: data.userId,
      pointsLimit: data.pointsLimit,
      updateTime: new Date(data.updateTime),
    };
  }

  async setPointsLimit(userId: string, pointsLimit: number): Promise<DailyPointsLimit> {
    const newLimit: DailyPointsLimit = {
      userId,
      pointsLimit,
      updateTime: new Date(),
    };
    
    const docRef = this.db.collection(this.collection).doc(userId);
    await docRef.set({
      userId: newLimit.userId,
      pointsLimit: newLimit.pointsLimit,
      updateTime: newLimit.updateTime.toISOString(),
    });
    
    return newLimit;
  }

}

// Export singleton instance
export const PointsLimitDatabase = new FirestorePointsLimitDatabase();