CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE availability_slots
    ADD CONSTRAINT availability_slots_no_overlap
    EXCLUDE USING gist (
        professional_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    );
