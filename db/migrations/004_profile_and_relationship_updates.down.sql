DROP INDEX IF EXISTS profile_relationship_active_pair_unique;

ALTER TABLE profile_relationship
    DROP CONSTRAINT IF EXISTS profile_relationship_type_length,
    DROP COLUMN IF EXISTS relationships_type;

ALTER TABLE profile_relationship
    ALTER COLUMN relationships_status TYPE TEXT
    USING relationships_status::TEXT;

DROP TYPE IF EXISTS relationships_status_enum;

ALTER TABLE profile_relationship
    ALTER COLUMN relationships_status TYPE relationships_enum
    USING CASE relationships_status
        WHEN 'request' THEN 'friend_request'::relationships_enum
        WHEN 'accepted' THEN 'friendship'::relationships_enum
        ELSE 'friendship'::relationships_enum
    END,
    RENAME COLUMN relationships_status TO status;

ALTER TABLE profile
    ADD COLUMN password_salt TEXT NOT NULL DEFAULT '',
    ALTER COLUMN profile_name SET NOT NULL,
    ALTER COLUMN surname SET NOT NULL,
    ALTER COLUMN gender SET NOT NULL,
    ADD CONSTRAINT profile_email_or_phone_check
        CHECK (email IS NOT NULL OR phone_number IS NOT NULL);

ALTER TABLE profile
    ALTER COLUMN password_salt DROP DEFAULT;

DROP TABLE IF EXISTS community_avatar;
