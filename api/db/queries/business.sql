-- name: CreateBusiness :one
INSERT INTO business (id, name, owner_id, country, currency)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetBusinessByOwnerID :one
SELECT *
FROM business
WHERE owner_id = $1;