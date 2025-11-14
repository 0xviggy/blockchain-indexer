#!/bin/bash
set -e

# This script runs during PostgreSQL initialization
# It's useful for setting up extensions and initial configuration

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Enable required extensions
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
    
    -- Grant necessary permissions
    GRANT ALL PRIVILEGES ON DATABASE indexer TO indexer;
    
    -- Log initialization
    SELECT 'Database initialized successfully' AS status;
EOSQL
