-- name: CreateUser :one
INSERT INTO users (email, first_name, last_name, password)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: UserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, expiry_at)
VALUES ($1, $2) RETURNING id;

-- name: SessionByUserID :one
SELECT * FROM sessions
WHERE user_id = $1 LIMIT 1;