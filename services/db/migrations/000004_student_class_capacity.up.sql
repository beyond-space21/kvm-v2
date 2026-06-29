ALTER TABLE users
    ADD COLUMN class_id UUID REFERENCES academic_classes(id) ON DELETE RESTRICT;

-- Existing students cannot satisfy class_id requirement without manual backfill.
DELETE FROM users WHERE role = 'student';

ALTER TABLE users
    ADD CONSTRAINT users_class_id_role_check
    CHECK ((role = 'student' AND class_id IS NOT NULL) OR (role != 'student' AND class_id IS NULL));

ALTER TABLE batches DROP COLUMN IF EXISTS capacity;

CREATE OR REPLACE FUNCTION check_enrollment_batch_offering()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM batches b
        WHERE b.id = NEW.batch_id AND b.offering_id = NEW.offering_id
    ) THEN
        RAISE EXCEPTION 'batch does not belong to offering';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enrollments_batch_offering_check
    BEFORE INSERT OR UPDATE ON enrollments
    FOR EACH ROW EXECUTE FUNCTION check_enrollment_batch_offering();
