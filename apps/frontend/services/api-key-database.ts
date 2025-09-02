import { Firestore } from '@google-cloud/firestore';

export interface ApiKeyBinding {
  api_key: string;                  // Primary key - document ID
  user_email: string;               // User's email address
  created_at: Date;                 // When the binding was created
}

class FirestoreApiKeyDatabase {
  private db: Firestore;
  private collection = 'api_key_bindings';

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

  async create(binding: Omit<ApiKeyBinding, 'created_at'>): Promise<ApiKeyBinding> {
    const newBinding: ApiKeyBinding = {
      ...binding,
      created_at: new Date(),
    };
    
    // Use transaction to enforce 3 key limit per user
    return await this.db.runTransaction(async (transaction) => {
      // Check current count for user
      const userKeysQuery = this.db.collection(this.collection)
        .where('user_email', '==', binding.user_email);
      const existingKeys = await transaction.get(userKeysQuery);
      
      if (existingKeys.size >= 3) {
        throw new Error('User already has maximum of 3 API keys');
      }
      
      // Create the new API key
      const docRef = this.db.collection(this.collection).doc(binding.api_key);
      transaction.set(docRef, {
        user_email: newBinding.user_email,
        created_at: newBinding.created_at.toISOString(),
      });
      
      return newBinding;
    });
  }

  async findByApiKey(apiKey: string): Promise<ApiKeyBinding | null> {
    const docRef = this.db.collection(this.collection).doc(apiKey);
    const doc = await docRef.get();
    
    if (!doc.exists) {
      return null;
    }
    
    const data = doc.data()!;
    return {
      api_key: apiKey,
      user_email: data.user_email,
      created_at: new Date(data.created_at),
    };
  }

  async findByUserEmail(userEmail: string): Promise<ApiKeyBinding[]> {
    const query = this.db.collection(this.collection)
      .where('user_email', '==', userEmail);
    
    const snapshot = await query.get();
    return snapshot.docs.map(doc => {
      const data = doc.data();
      return {
        api_key: doc.id,
        user_email: data.user_email,
        created_at: new Date(data.created_at),
      };
    });
  }

  async deleteApiKey(apiKey: string): Promise<void> {
    const docRef = this.db.collection(this.collection).doc(apiKey);
    await docRef.delete();
  }
}

export const ApiKeyDatabase = new FirestoreApiKeyDatabase();