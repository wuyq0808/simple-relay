const { Firestore } = require('@google-cloud/firestore');

async function debugHourlyAggregates() {
  const db = new Firestore({
    projectId: 'simple-relay-468808',
    databaseId: 'simple-relay-db-staging',
  });

  const userEmail = 'wuyongqi1988@gmail.com';
  
  // Get current 24-hour window (8pm-8pm UTC)
  const now = new Date();
  let windowStart;
  if (now.getUTCHours() >= 20) {
    windowStart = new Date(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), 20, 0, 0, 0);
  } else {
    const yesterday = new Date(now);
    yesterday.setUTCDate(now.getUTCDate() - 1);
    windowStart = new Date(yesterday.getUTCFullYear(), yesterday.getUTCMonth(), yesterday.getUTCDate(), 20, 0, 0, 0);
  }
  const windowEnd = new Date(windowStart);
  windowEnd.setUTCHours(windowStart.getUTCHours() + 24);

  console.log(`Checking hourly aggregates for ${userEmail}`);
  console.log(`Window: ${windowStart.toISOString()} to ${windowEnd.toISOString()}`);
  console.log('==========================================');

  const query = db.collection('hourly_aggregates')
    .where('user_id', '==', userEmail)
    .where('hour', '>=', windowStart)
    .where('hour', '<', windowEnd);

  const snapshot = await query.get();
  
  let totalPointsSum = 0;
  let modelPointsSum = 0;

  snapshot.forEach(doc => {
    const data = doc.data();
    const docId = doc.id;
    const hour = data.hour?.toDate() || new Date(data.hour);
    
    console.log(`\nDocument: ${docId}`);
    console.log(`Hour: ${hour.toISOString()}`);
    console.log(`Simple total_points: ${data.total_points || 0}`);
    
    // Extract all model_usage.{model}.total_points fields
    let docModelSum = 0;
    const modelFields = [];
    
    for (const [key, value] of Object.entries(data)) {
      if (key.startsWith('model_usage.') && key.endsWith('.total_points')) {
        const model = key.split('.')[1];
        modelFields.push({ model, points: value });
        docModelSum += value;
      }
    }
    
    console.log(`Model usage fields:`);
    modelFields.forEach(({ model, points }) => {
      console.log(`  ${model}: ${points} points`);
    });
    console.log(`Sum of model points: ${docModelSum}`);
    console.log(`Difference: ${(data.total_points || 0) - docModelSum}`);
    
    totalPointsSum += (data.total_points || 0);
    modelPointsSum += docModelSum;
  });

  console.log('\n==========================================');
  console.log(`TOTALS FOR 24-HOUR WINDOW:`);
  console.log(`Sum of all simple total_points: ${totalPointsSum}`);
  console.log(`Sum of all model_usage.{model}.total_points: ${modelPointsSum}`);
  console.log(`Overall difference: ${totalPointsSum - modelPointsSum}`);
  
  if (Math.abs(totalPointsSum - modelPointsSum) > 0.001) {
    console.log('🚨 DATA INCONSISTENCY CONFIRMED!');
  } else {
    console.log('✅ Data appears consistent');
  }
}

debugHourlyAggregates().catch(console.error);