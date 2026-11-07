-- The cron jobs a project declares (v0.6): [{command, schedule}], as the
-- agent reports them. Each names one of the project's commands.
ALTER TABLE projects ADD COLUMN cron TEXT NOT NULL DEFAULT '[]';
