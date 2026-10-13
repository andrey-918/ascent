ALTER TABLE diary_entries
    ADD CONSTRAINT diary_entries_id_user_id_key UNIQUE (id, user_id);

ALTER TABLE goals
    ADD CONSTRAINT goals_id_user_id_key UNIQUE (id, user_id);

CREATE TABLE financial_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL,
    diary_entry_id UUID NOT NULL,
    event_type VARCHAR(16) NOT NULL CHECK (event_type IN ('deposit', 'withdrawal')),
    amount_kopecks BIGINT NOT NULL CHECK (amount_kopecks > 0),
    currency CHAR(3) NOT NULL DEFAULT 'RUB' CHECK (currency = 'RUB'),
    source_text TEXT NOT NULL CHECK (length(trim(source_text)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT financial_events_goal_owner_fkey
        FOREIGN KEY (goal_id, user_id) REFERENCES goals(id, user_id) ON DELETE CASCADE,
    CONSTRAINT financial_events_diary_entry_owner_fkey
        FOREIGN KEY (diary_entry_id, user_id) REFERENCES diary_entries(id, user_id) ON DELETE RESTRICT
);

CREATE INDEX financial_events_goal_created_at_idx
    ON financial_events (goal_id, created_at DESC, id DESC);
