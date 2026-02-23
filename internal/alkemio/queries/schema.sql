-- Schema definition for SQLC code generation
-- This mirrors the existing Alkemio "user" table (read-only access).
-- user.id is also the actorId (FK to actor table).

CREATE TABLE "user" (
    "id" UUID PRIMARY KEY,
    "authenticationID" UUID UNIQUE NOT NULL
);
