CREATE TABLE IF NOT EXISTS task_templates (
    id                 BIGSERIAL    PRIMARY KEY,
    title              TEXT         NOT NULL,
    description        TEXT         NOT NULL DEFAULT '',
    periodicity_type   TEXT         NOT NULL,
    periodicity_params JSONB        NOT NULL DEFAULT '{}',
    start_date         DATE         NOT NULL,
    end_date           DATE,
    is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_templates_is_active ON task_templates (is_active) WHERE is_active = TRUE;

ALTER TABLE tasks
    ADD COLUMN template_id    BIGINT REFERENCES task_templates(id) ON DELETE SET NULL,
    ADD COLUMN scheduled_date DATE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_template_date
    ON tasks (template_id, scheduled_date) WHERE template_id IS NOT NULL;
