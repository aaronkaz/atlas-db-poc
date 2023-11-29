-- name: CreateUser :one
INSERT INTO users (
  id, name, title
) VALUES (
  $1, $2, $3
)
RETURNING *;