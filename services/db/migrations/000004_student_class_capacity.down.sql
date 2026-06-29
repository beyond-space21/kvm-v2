DROP TRIGGER IF EXISTS enrollments_batch_offering_check ON enrollments;
DROP FUNCTION IF EXISTS check_enrollment_batch_offering();

ALTER TABLE batches ADD COLUMN capacity INT CHECK (capacity IS NULL OR capacity > 0);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_class_id_role_check;
ALTER TABLE users DROP COLUMN IF EXISTS class_id;
