-- name: GetUserByAuthenticationID :one
SELECT "id"
FROM "user"
WHERE "authenticationID" = $1;
