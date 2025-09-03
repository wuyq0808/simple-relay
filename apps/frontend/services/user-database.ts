import { Firestore } from '@google-cloud/firestore';

export interface ApiKey {
  api_key: string;
  created_at: Date;
}

export interface User {
  email: string;                    // Primary key
  created_at: Date;                 // Account creation timestamp
  last_login: Date | null;          // Last login timestamp
  verification_token: string | null; // Token for email verification
  verification_expires_at: Date | null; // Expiration time for verification token
  api_keys: ApiKey[];               // Array of API keys for this user
  api_enabled: boolean;             // Whether API access is enabled for this user
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

  async create(user: Omit<User, 'created_at' | 'api_keys' | 'api_enabled'>): Promise<User> {
    const newUser: User = {
      ...user,
      created_at: new Date(),
      api_keys: [],
      api_enabled: false,
    };
    
    const docRef = this.db.collection(this.collection).doc(user.email);
    await docRef.set({
      ...newUser,
      created_at: newUser.created_at.toISOString(),
      last_login: newUser.last_login?.toISOString() || null,
      verification_expires_at: newUser.verification_expires_at?.toISOString() || null,
      api_keys: newUser.api_keys.map(key => ({
        api_key: key.api_key,
        created_at: key.created_at.toISOString(),
      })),
      api_enabled: newUser.api_enabled,
    });
    
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
      api_keys: (data.api_keys || []).map((key: any) => ({
        api_key: key.api_key,
        created_at: new Date(key.created_at),
      })),
      api_enabled: data.api_enabled !== undefined ? data.api_enabled : false,
    };
  }

  async updateUser(email: string, updates: Partial<Omit<User, 'email'>>): Promise<User | null> {
    const docRef = this.db.collection(this.collection).doc(email);
    
    const firestoreUpdates: any = {
      ...updates,
      last_login: updates.last_login?.toISOString() || null,
      verification_expires_at: updates.verification_expires_at?.toISOString() || null,
    };
    
    if (updates.api_keys) {
      firestoreUpdates.api_keys = updates.api_keys.map(key => ({
        api_key: key.api_key,
        created_at: key.created_at.toISOString(),
      }));
    }
    
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

  // API Key management methods
  async addApiKey(email: string, apiKey: string): Promise<User | null> {
    const user = await this.findByEmail(email);
    if (!user) return null;
    
    // Check if user already has 3 API keys
    if (user.api_keys.length >= 3) {
      throw new Error('User already has maximum of 3 API keys');
    }
    
    const newApiKey: ApiKey = {
      api_key: apiKey,
      created_at: new Date(),
    };
    
    const updatedApiKeys = [...user.api_keys, newApiKey];
    return this.updateUser(email, { api_keys: updatedApiKeys });
  }

  async removeApiKey(email: string, apiKey: string): Promise<User | null> {
    const user = await this.findByEmail(email);
    if (!user) return null;
    
    const updatedApiKeys = user.api_keys.filter(key => key.api_key !== apiKey);
    return this.updateUser(email, { api_keys: updatedApiKeys });
  }

  async findUserByApiKey(apiKey: string): Promise<User | null> {
    const query = this.db.collection(this.collection)
      .where('api_keys', 'array-contains-any', [{ api_key: apiKey }]);
    
    const snapshot = await query.get();
    
    if (snapshot.empty) {
      // Fallback: search through all users (less efficient but more reliable)
      const allUsersSnapshot = await this.db.collection(this.collection).get();
      
      for (const doc of allUsersSnapshot.docs) {
        const userData = doc.data();
        const apiKeys = userData.api_keys || [];
        
        if (apiKeys.some((key: any) => key.api_key === apiKey)) {
          return {
            email: userData.email,
            created_at: new Date(userData.created_at),
            last_login: userData.last_login ? new Date(userData.last_login) : null,
            verification_token: userData.verification_token || null,
            verification_expires_at: userData.verification_expires_at ? new Date(userData.verification_expires_at) : null,
            api_keys: apiKeys.map((key: any) => ({
              api_key: key.api_key,
              created_at: new Date(key.created_at),
            })),
            enabled: userData.enabled !== undefined ? userData.enabled : true,
          };
        }
      }
      return null;
    }
    
    const doc = snapshot.docs[0];
    const data = doc.data();
    return {
      email: data.email,
      created_at: new Date(data.created_at),
      last_login: data.last_login ? new Date(data.last_login) : null,
      verification_token: data.verification_token || null,
      verification_expires_at: data.verification_expires_at ? new Date(data.verification_expires_at) : null,
      api_keys: (data.api_keys || []).map((key: any) => ({
        api_key: key.api_key,
        created_at: new Date(key.created_at),
      })),
      api_enabled: data.api_enabled !== undefined ? data.api_enabled : false,
    };
  }

}

// Export singleton instance
export const UserDatabase = new FirestoreUserDatabase();