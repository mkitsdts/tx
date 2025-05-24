CREATE EXTENSION IF NOT EXISTS vector;
CREATE DATABASE IF NOT EXISTS tx_test;

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    likes TEXT NULL,
    like_embedding vector(384) NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建向量索引
CREATE INDEX IF NOT EXISTS like_embedding_idx ON users USING ivfflat (like_embedding vector_cosine_ops) WITH (lists = 100);

-- 通过命令
-- psql -h localhost -p 5432 -U mkitsdts -d tx_test