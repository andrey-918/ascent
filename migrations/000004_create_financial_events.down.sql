DROP TABLE financial_events;

ALTER TABLE goals
    DROP CONSTRAINT goals_id_user_id_key;

ALTER TABLE diary_entries
    DROP CONSTRAINT diary_entries_id_user_id_key;
