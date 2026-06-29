ALTER TABLE subject_offerings
    DROP CONSTRAINT IF EXISTS subject_offerings_academic_year_id_class_id_subject_id_key;

ALTER TABLE subject_offerings
    DROP CONSTRAINT IF EXISTS subject_offerings_academic_year_id_fkey;

ALTER TABLE subject_offerings
    DROP COLUMN IF EXISTS academic_year_id;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'subject_offerings_class_id_subject_id_key'
    ) THEN
        ALTER TABLE subject_offerings
            ADD CONSTRAINT subject_offerings_class_id_subject_id_key UNIQUE (class_id, subject_id);
    END IF;
END $$;
