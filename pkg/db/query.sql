-- name: CreateUser :one
INSERT INTO users (email, first_name, last_name, password)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: UserByEmail :one
SELECT * FROM users
WHERE email = $1 AND active = true LIMIT 1;

-- name: UpdateLastLogin :exec
UPDATE users SET last_login = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, expiry_at)
VALUES ($1, $2) RETURNING id;

-- name: SessionByUserID :one
SELECT * FROM sessions
WHERE user_id = $1 LIMIT 1;

-- name: UserBySessionID :one
SELECT * FROM users
INNER JOIN sessions ON users.id = sessions.user_id
WHERE users.active = true AND sessions.id = $1
ORDER BY sessions.expiry_at DESC LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteExpiredSessionsWithThreshold :execrows
DELETE FROM sessions WHERE sessions.expiry_at < (NOW() - interval '180 minute');