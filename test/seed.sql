CREATE TABLE users
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL,
    password   TEXT NOT NULL,
    active     BOOLEAN DEFAULT FALSE,
    last_login TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL
);

CREATE TABLE sessions
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    expiry_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE iga_sessions
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    project_name TEXT NOT NULL,
    context_data JSONB NOT NULL DEFAULT '{}'::jsonb,    -- Stores stakeholder_desc, project_desc, etc.
    system_metadata JSONB NULL,                         -- Stores LLM settings, etc.
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL
);

CREATE TABLE iga_script_questions
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    iga_session_id UUID REFERENCES iga_sessions(id) ON DELETE CASCADE,
    readable_id VARCHAR(16) NOT NULL,
    question_text TEXT NOT NULL,
    note_text TEXT NULL,
    sort_order SMALLINT NOT NULL,
    metadata JSONB NULL
);

CREATE INDEX ON iga_script_questions (iga_session_id, sort_order);
CREATE UNIQUE INDEX ON iga_script_questions (iga_session_id, readable_id);

CREATE TABLE iga_chat_logs
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    iga_session_id UUID REFERENCES iga_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    message_text TEXT NOT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX ON iga_chat_logs (iga_session_id, created_at)