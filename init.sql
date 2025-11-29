-- Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Create articles table with geography columns
CREATE TABLE IF NOT EXISTS articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    publication_date TIMESTAMP NOT NULL,
    source_name VARCHAR(255) NOT NULL,
    category TEXT[] NOT NULL,
    relevance_score FLOAT NOT NULL CHECK (relevance_score >= 0 AND relevance_score <= 1),
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    summary TEXT,
    description_vector VECTOR(1536)
);

-- Create user_events table with geography column
CREATE TABLE IF NOT EXISTS user_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    article_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('view', 'click')),
    timestamp TIMESTAMP NOT NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for articles table
-- GIN index for array category field
CREATE INDEX IF NOT EXISTS idx_articles_category ON articles USING GIN(category);

-- B-tree index for source_name
CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source_name);

-- B-tree index for relevance_score
CREATE INDEX IF NOT EXISTS idx_articles_score ON articles(relevance_score DESC);

-- B-tree index for publication_date
CREATE INDEX IF NOT EXISTS idx_articles_publication_date ON articles(publication_date DESC);

-- Index for latitude/longitude queries
CREATE INDEX IF NOT EXISTS idx_articles_lat_lon ON articles(latitude, longitude);

-- Create indexes for user_events table
-- Composite index for article_id and timestamp queries
CREATE INDEX IF NOT EXISTS idx_user_events_article ON user_events(article_id, timestamp DESC);

-- Index for latitude/longitude queries
CREATE INDEX IF NOT EXISTS idx_user_events_lat_lon ON user_events(latitude, longitude);

-- B-tree index for timestamp queries
CREATE INDEX IF NOT EXISTS idx_user_events_timestamp ON user_events(timestamp DESC);

-- B-tree index for user_id queries
CREATE INDEX IF NOT EXISTS idx_user_events_user ON user_events(user_id);

CREATE INDEX articles_geo_idx ON articles USING GIST (
    (ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography)
);

