-- name: GetUserByAuthenticationID :one
SELECT "id", "agentId"
FROM "user"
WHERE "authenticationID" = $1;
