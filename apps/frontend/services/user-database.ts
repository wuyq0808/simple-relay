import { Firestore } from '@google-cloud/firestore';
import { CostLimitDatabase } from './cost-limit-database.js';

export interface User {
  email: string;                    // Primary key
  created_at: Date;                 // Account creation timestamp
  last_login: Date | null;          // Last login timestamp
  verification_token: string | null; // Token for email verification
  verification_expires_at: Date | null; // Expiration time for verification token
  api_enabled: boolean;             // Whether API access is enabled for this user
  access_approval_pending?: boolean; // Whether the user has a pending access approval request
}

class FirestoreUserDatabase {
  private db: Firestore;
  private collection = 'users';

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

  async create(user: Omit<User, 'created_at' | 'api_enabled'>): Promise<User> {
    const newUser: User = {
      ...user,
      created_at: new Date(),
      api_enabled: true,
    };
    
    // Create user document
    const docRef = this.db.collection(this.collection).doc(user.email);
    await docRef.set({
      ...newUser,
      created_at: newUser.created_at.toISOString(),
      last_login: newUser.last_login?.toISOString() || null,
      verification_expires_at: newUser.verification_expires_at?.toISOString() || null,
      api_enabled: newUser.api_enabled,
    });
    
    // Set initial cost limit of 0.05 for new users
    await CostLimitDatabase.setCostLimit(user.email, 0.05);
    
    return newUser;
  }

  async findByEmail(email: string): Promise<User | null> {
    const docRef = this.db.collection(this.collection).doc(email);
    const doc = await docRef.get();
    
    if (!doc.exists) {
      return null;
    }
    
    const data = doc.data()!;
    return {
      email: data.email,
      created_at: new Date(data.created_at),
      last_login: data.last_login ? new Date(data.last_login) : null,
      verification_token: data.verification_token || null,
      verification_expires_at: data.verification_expires_at ? new Date(data.verification_expires_at) : null,
      api_enabled: data.api_enabled !== undefined ? data.api_enabled : false,
      access_approval_pending: data.access_approval_pending || false,
    };
  }

  async updateUser(email: string, updates: Partial<Omit<User, 'email'>>): Promise<User | null> {
    const docRef = this.db.collection(this.collection).doc(email);
    
    const firestoreUpdates: Record<string, unknown> = {
      ...updates,
      last_login: updates.last_login?.toISOString() || null,
      verification_expires_at: updates.verification_expires_at?.toISOString() || null,
    };
    
    await docRef.update(firestoreUpdates);
    return this.findByEmail(email);
  }

  async verifyUser(email: string): Promise<User | null> {
    const docRef = this.db.collection(this.collection).doc(email);
    
    await docRef.update({
      verification_token: null,
      verification_expires_at: null,
      last_login: new Date().toISOString(),
    });
    
    return this.findByEmail(email);
  }

  async updateLastLogin(email: string): Promise<User | null> {
    const docRef = this.db.collection(this.collection).doc(email);
    
    await docRef.update({
      last_login: new Date().toISOString(),
    });
    
    return this.findByEmail(email);
  }

  // Helper method to check if verification token is valid
  isVerificationTokenValid(user: User): boolean {
    if (!user.verification_token || !user.verification_expires_at) {
      return false;
    }
    return user.verification_expires_at > new Date();
  }


}

// Export singleton instance
export const UserDatabase = new FirestoreUserDatabase();