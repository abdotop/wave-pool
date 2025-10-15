-- name: GetUserByID :one
SELECT * FROM "users" WHERE id = $1;

-- name: GetUserByPhone :one
SELECT * FROM "users" WHERE phone = $1;

-- name: CreateUser :one
INSERT INTO "users" (id, phone, pin_hash)
VALUES ($1, $2, $3)
RETURNING *;