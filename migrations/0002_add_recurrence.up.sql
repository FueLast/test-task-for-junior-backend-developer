ALTER TABLE tasks
ADD COLUMN recurrence_type TEXT NOT NULL DEFAULT '',
ADD COLUMN recurrence_data JSONB;