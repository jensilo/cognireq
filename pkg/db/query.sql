---
--- User and Session Management ---
---


-- name: CreateUser :one
INSERT INTO users (email, first_name, last_name, password)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ActivateUserByID :execrows
UPDATE users SET active = true WHERE id = $1;

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
ORDER BY updated_at DESC, created_at DESC;

-- name: CreateIGASession :one
INSERT INTO iga_sessions (user_id, project_name, context_data, system_metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: IGASessionByID :one
SELECT * FROM iga_sessions
WHERE id = $2 AND user_id = $1 LIMIT 1;

-- name: PatchIGASession :one
UPDATE iga_sessions
SET
    project_name = COALESCE(sqlc.narg('project_name'), project_name),
    context_data = COALESCE(sqlc.narg('context_data'), context_data),
    system_metadata = COALESCE(sqlc.narg('system_metadata'), system_metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: TouchIGASessionUpdatedAt :one
UPDATE iga_sessions
SET updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND user_id = $1
RETURNING *;


---
--- IGA Chat Log Management ---
---


-- name: IGASessionExistsForUser :one
SELECT EXISTS(
    SELECT 1 FROM iga_sessions
    WHERE id = $2 AND user_id = $1
);

-- name: IGAChatLogsBySessionID :many
SELECT iga_chat_logs.* FROM iga_chat_logs
INNER JOIN iga_sessions ON iga_chat_logs.iga_session_id = iga_sessions.id
WHERE iga_sessions.id = $2 AND iga_sessions.user_id = $1
ORDER BY iga_chat_logs.created_at;

-- name: CreateIGAChatLog :one
INSERT INTO iga_chat_logs (iga_session_id, role, message_text, metadata)
SELECT $2, $3, $4, $5
FROM iga_sessions
WHERE iga_sessions.id = $2 AND iga_sessions.user_id = $1
RETURNING *;


---
--- IGA Questions Management ---
---

-- name: IGAScriptQuestionsBySessionID :many
SELECT iga_script_questions.* FROM iga_script_questions
INNER JOIN iga_sessions ON iga_script_questions.iga_session_id = iga_sessions.id
WHERE iga_sessions.user_id = $1 AND iga_sessions.id = $2
ORDER BY iga_script_questions.sort_order;

-- name: IGAScriptQuestionIDsBySessionID :many
SELECT iga_script_questions.id FROM iga_script_questions
INNER JOIN iga_sessions ON iga_script_questions.iga_session_id = iga_sessions.id
WHERE iga_sessions.user_id = $1 AND iga_sessions.id = $2;

-- name: IGAScriptQuestionByID :one
SELECT iga_script_questions.* FROM iga_script_questions
INNER JOIN iga_sessions ON iga_script_questions.iga_session_id = iga_sessions.id
WHERE iga_script_questions.id = sqlc.arg('iga_script_question_id') AND iga_sessions.id = sqlc.arg('iga_session_id') AND iga_sessions.user_id = sqlc.arg('user_id');

-- name: CreateIGAScriptQuestion :one
INSERT INTO iga_script_questions (iga_session_id, readable_id, question_text, note_text, sort_order, metadata)
SELECT $2, $3, $4, $5, $6, $7
FROM iga_sessions
WHERE iga_sessions.id = $2 AND iga_sessions.user_id = $1
RETURNING *;

-- name: IGAScriptQuestionExistsByReadableID :one
SELECT EXISTS(
    SELECT 1 FROM iga_script_questions
    WHERE iga_session_id = $1 AND readable_id = $2
);

-- name: PatchIGAScriptQuestion :one
UPDATE iga_script_questions
SET
    readable_id = COALESCE(sqlc.narg('readable_id'), readable_id),
    question_text = COALESCE(sqlc.narg('question_text'), question_text),
    note_text = COALESCE(sqlc.narg('note_text'), note_text),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order),
    metadata = COALESCE(sqlc.narg('metadata'), metadata)
WHERE
    iga_script_questions.id = sqlc.arg('id') AND iga_session_id = sqlc.arg('iga_session_id')
    AND EXISTS (SELECT 1 FROM iga_sessions AS sess WHERE sess.id = sqlc.arg('iga_session_id') AND sess.user_id = sqlc.arg('user_id'))
RETURNING *;

-- name: DeleteIGAScriptQuestion :execrows
DELETE FROM iga_script_questions
WHERE
    iga_script_questions.id = $3 AND iga_session_id = $2
    AND EXISTS (SELECT 1 FROM iga_sessions AS sess WHERE sess.id = iga_script_questions.iga_session_id AND user_id = $1);

-- name: ReorderIGAScriptQuestions :batchexec
UPDATE iga_script_questions
SET sort_order = $2
WHERE id = $1;

