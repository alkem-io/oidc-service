-- Schema definition for SQLC code generation
-- This mirrors the existing Alkemio "user" table (read-only access)

CREATE TABLE "user" (
    "id" UUID PRIMARY KEY,
    "authenticationID" UUID UNIQUE NOT NULL,
    "agentId" UUID NOT NULL
);
