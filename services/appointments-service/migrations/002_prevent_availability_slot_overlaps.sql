CREATE EXTENSION IF NOT EXISTS btree_gist;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'availability_slots_no_overlap'
    ) THEN
        ALTER TABLE availability_slots
            ADD CONSTRAINT availability_slots_no_overlap
            EXCLUDE USING gist (
                professional_id WITH =,
                tstzrange(start_time, end_time, '[)') WITH &&
            );
    END IF;
END $$;
