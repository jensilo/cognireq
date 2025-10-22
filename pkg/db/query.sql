---
--- User and Session Management ---
---


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


---
--- IGA Session Management ---
---


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


---
--- IGA Chat Log Management ---
---


-- name: IGAChatLogsBySessionID :many
SELECT * FROM iga_chat_logs
INNER JOIN iga_sessions ON iga_chat_logs.iga_session_id = iga_sessions.id
WHERE iga_sessions.user_id = $1 AND iga_sessions.id = $1
ORDER BY iga_chat_logs.created_at DESC;

-- name: CreateIGAChatLog :one
INSERT INTO iga_chat_logs (iga_session_id, role, message_text, metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;


---
--- IGA Questions Management ---
---


-- name: IGAScriptQuestionsBySessionID :many
SELECT * FROM iga_script_questions
INNER JOIN iga_sessions ON iga_script_questions.iga_session_id = iga_sessions.id
WHERE iga_sessions.user_id = $1 AND iga_sessions.id = $1
ORDER BY iga_script_questions.sort_order;

-- name: CreateIGAScriptQuestion :one
INSERT INTO iga_script_questions (iga_session_id, readable_id, question_text, note_text, sort_order, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: PatchIGAScriptQuestion :one
WITH t AS (
    SELECT t_sess.user_id AS user_id
    FROM iga_sessions AS t_sess
    INNER JOIN iga_script_questions AS t_ques ON t_sess.id = t_ques.iga_session_id
)
UPDATE iga_script_questions
SET
    readable_id = COALESCE($3, readable_id),
    question_text = COALESCE($4, question_text),
    note_text = COALESCE($5, note_text),
    sort_order = COALESCE($6, sort_order),
    metadata = COALESCE($7, metadata)
WHERE t.user_id = $1 AND iga_session_id = $2
RETURNING *;

-- name: DeleteIGAScriptQuestion :exec
WITH t AS (
    SELECT t_sess.user_id AS user_id
    FROM iga_sessions AS t_sess
             INNER JOIN iga_script_questions AS t_ques ON t_sess.id = t_ques.iga_session_id
)
DELETE FROM iga_script_questions
WHERE t.user_id = $1 AND id = $2;
