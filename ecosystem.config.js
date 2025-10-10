module.exports = {
  apps: [
    {
      name: 'sql-studio-backend',
      script: './dist/server.js',
      instances: process.env.PM2_INSTANCES || 'max',
      exec_mode: 'cluster',
      max_memory_restart: '512M',
      node_args: '--max-old-space-size=512',
      env: {
        NODE_ENV: 'development',
        PORT: 3000,
        LOG_LEVEL: 'debug'
      },
      env_production: {
        NODE_ENV: 'production',
        PORT: 3000,
        LOG_LEVEL: 'info',
        // Security
        HELMET_ENABLED: true,
        CORS_ENABLED: true,
        RATE_LIMIT_ENABLED: true,
        COMPRESSION_ENABLED: true,
        // Performance
        CLUSTER_MODE: true,
        KEEP_ALIVE_TIMEOUT: 65000,
        HEADERS_TIMEOUT: 66000,
        // Monitoring
        PROMETHEUS_ENABLED: true,
        PROMETHEUS_PORT: 9090,
        HEALTH_CHECK_ENABLED: true
      },
      error_file: './logs/err.log',
      out_file: './logs/out.log',
      log_file: './logs/combined.log',
      time: true,
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      // Auto restart configuration
      watch: false,
      ignore_watch: ['node_modules', 'logs', 'data'],
      restart_delay: 4000,
      max_restarts: 10,
      min_uptime: '10s',
      // Health monitoring
      kill_timeout: 5000,
      listen_timeout: 10000,
      // Performance monitoring
      monitoring: true,
      pmx: false
    }
  ],
  deploy: {
    production: {
      user: 'deploy',
      host: ['production-server'],
      ref: 'origin/main',
      repo: 'git@github.com:your-org/sql-studio.git',
      path: '/var/www/sql-studio',
      'post-deploy': 'npm install && npm run build && pm2 reload ecosystem.config.js --env production',
      'pre-setup': 'apt update && apt install git -y'
    },
    staging: {
      user: 'deploy',
      host: ['staging-server'],
      ref: 'origin/develop',
      repo: 'git@github.com:your-org/sql-studio.git',
      path: '/var/www/sql-studio-staging',
      'post-deploy': 'npm install && npm run build && pm2 reload ecosystem.config.js --env staging'
    }
  }
};