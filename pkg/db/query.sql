--
--  --- User and Session Management ---
--

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

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE sessions.expiry_at < NOW();

--
--  --- IGA Session Management ---
--

-- name: IGASessionsByUser :many
SELECT * FROM iga_sessions
WHERE user_id = $1
ORDER BY updated_at DESC;

-- name: CreateIGASession :one
INSERT INTO iga_sessions (user_id, project_name, context_data, system_metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: IGASessionByID :one
SELECT * FROM iga_sessions
WHERE user_id = $1 AND id = $2 LIMIT 1;

-- name: PatchIGASession :one
UPDATE iga_sessions
SET
    project_name = COALESCE($3, project_name),
    context_data = COALESCE($4, context_data),
    system_metadata = COALESCE($5, system_metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND id = $2
RETURNING *;

--
--  --- IGA Chat Log Management ---
--

--
--  --- IGA Questions Management ---
--