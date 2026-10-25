CREATE TYPE relationships_enum AS ENUM (
    'friendship',
    'blocked',
    'friend_request'
);

DROP INDEX IF EXISTS profile_relationship_active_pair_unique;

ALTER TABLE profile_relationship
    ALTER COLUMN relationships_status TYPE relationships_enum
    USING CASE
        WHEN relationships_status = 'request'
            THEN 'friend_request'::relationships_enum
        WHEN relationships_status = 'accepted'
             AND relationships_type = 'blocked'
            THEN 'blocked'::relationships_enum
        WHEN relationships_status = 'accepted'
            THEN 'friendship'::relationships_enum
        ELSE 'friendship'::relationships_enum
    END;

-- RENAME COLUMN выполняется отдельной командой ALTER TABLE
ALTER TABLE profile_relationship
    RENAME COLUMN relationships_status TO status;

ALTER TABLE profile_relationship
    DROP CONSTRAINT IF EXISTS profile_relationship_type_length,
    DROP COLUMN IF EXISTS relationships_type;

DROP TYPE IF EXISTS relationships_status_enum;

-- Восстановаем NOT NULL значения
UPDATE profile
SET profile_name = COALESCE(profile_name, nickname),
    surname = COALESCE(surname, 'Unknown'),
    gender = COALESCE(gender, 'other'::gender_enum),
    email = CASE
        WHEN email IS NULL AND phone_number IS NULL
            THEN 'rollback_' || id::TEXT || '@invalid.local'
        ELSE email
    END
WHERE profile_name IS NULL
   OR surname IS NULL
   OR gender IS NULL
   OR (email IS NULL AND phone_number IS NULL);

ALTER TABLE profile
    ADD COLUMN password_salt TEXT NOT NULL DEFAULT '',
    ALTER COLUMN profile_name SET NOT NULL,
    ALTER COLUMN surname SET NOT NULL,
    ALTER COLUMN gender SET NOT NULL,
    ADD CONSTRAINT profile_email_or_phone_check
        CHECK (email IS NOT NULL OR phone_number IS NOT NULL),

    ADD CONSTRAINT profile_nickname_unique UNIQUE (nickname),
    ADD CONSTRAINT profile_email_unique UNIQUE (email),
    ADD CONSTRAINT profile_phone_unique UNIQUE (phone_number);


DROP INDEX profile_nickname_unique;
DROP INDEX profile_email_unique;
DROP INDEX profile_phone_unique;


ALTER TABLE profile
    ALTER COLUMN password_salt DROP DEFAULT;

DROP TABLE IF EXISTS community_avatar;
