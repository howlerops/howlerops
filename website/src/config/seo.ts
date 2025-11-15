export const seoConfig = {
  title: 'Howlerops - Modern SQL Editor for Teams',
  description: 'Write, share, and sync SQL queries across your team. Connect to any database with ease, collaborate in real-time, and boost your team\'s productivity.',
  keywords: [
    'sql editor',
    'database management',
    'query tool',
    'team collaboration',
    'multi-database support',
    'cloud sync',
    'ai sql assistant'
  ],
  ogImage: '/og-image.png',
  twitterCard: 'summary_large_image',
  canonical: 'https://sqlstudio.app'
}

export const analyticsConfig = {
  providers: ['google-analytics', 'plausible'],
  trackEvents: [
    'signup',
    'trial_start',
    'feature_explore',
    'query_executed',
    'connection_created',
    'documentation_view'
  ]
}