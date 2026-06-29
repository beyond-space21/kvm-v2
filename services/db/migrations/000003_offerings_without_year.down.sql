ALTER TABLE subject_offerings
    DROP CONSTRAINT IF EXISTS subject_offerings_class_id_subject_id_key;

ALTER TABLE subject_offerings
    ADD COLUMN academic_year_id UUID REFERENCES academic_years(id) ON DELETE CASCADE;

UPDATE subject_offerings
SET academic_year_id = (
    SELECT id FROM academic_years ORDER BY is_active DESC, start_date DESC LIMIT 1
)
WHERE academic_year_id IS NULL;

ALTER TABLE subject_offerings
    ALTER COLUMN academic_year_id SET NOT NULL;

ALTER TABLE subject_offerings
    ADD CONSTRAINT subject_offerings_academic_year_id_class_id_subject_id_key UNIQUE (academic_year_id, class_id, subject_id);
