import { Firestore } from '@google-cloud/firestore';

export interface AppConfig {
  key: string;
  value: boolean | string | number;
  description?: string;
  updated_at: Date;
}

type ConfigValue = boolean | string | number;

function isValidConfigValue(value: any): value is ConfigValue {
  return typeof value === 'boolean' || 
         typeof value === 'string' || 
         typeof value === 'number';
}

interface CacheEntry {
  configs: Record<string, ConfigValue>;
  timestamp: number;
}

class FirestoreConfigService {
  private db: Firestore;
  private collection = 'app_config';
  private cache: CacheEntry | null = null;
  private readonly CACHE_DURATION_MS = 60 * 1000; // 1 minute

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

  private isCacheValid(): boolean {
    if (!this.cache) return false;
    const now = Date.now();
    return (now - this.cache.timestamp) < this.CACHE_DURATION_MS;
  }

  private async loadAllConfigs(): Promise<Record<string, ConfigValue>> {
    const snapshot = await this.db.collection(this.collection).get();
    const configs: Record<string, ConfigValue> = {};
    
    snapshot.forEach(doc => {
      const data = doc.data();
      if (data && isValidConfigValue(data.value)) {
        configs[doc.id] = data.value;
      }
    });
    
    // Update cache
    this.cache = {
      configs,
      timestamp: Date.now()
    };
    
    return configs;
  }

  async getConfig(key: string): Promise<ConfigValue | null> {
    let configs: Record<string, ConfigValue>;
    
    if (this.isCacheValid()) {
      configs = this.cache!.configs;
    } else {
      configs = await this.loadAllConfigs();
    }
    
    return configs[key] ?? null;
  }
}

export const ConfigService = new FirestoreConfigService();
