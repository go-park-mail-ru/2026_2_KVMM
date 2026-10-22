-- Поля профиля стали необязательными (переход на минимальную регистрацию)
ALTER TABLE profile
    ALTER COLUMN profile_name DROP NOT NULL,
    ALTER COLUMN surname DROP NOT NULL,
    ALTER COLUMN gender DROP NOT NULL,
    DROP COLUMN password_salt;

-- Аватары сообществ
CREATE TABLE community_avatar (
    community_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (community_id, media_id),

    FOREIGN KEY (community_id)
        REFERENCES community (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (media_id)
        REFERENCES media (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

-- Старый статус связи преобразуется в статус и тип
ALTER TABLE profile_relationship
    RENAME COLUMN status TO relationships_status;

ALTER TABLE profile_relationship
    ALTER COLUMN relationships_status TYPE TEXT
    USING relationships_status::TEXT;

ALTER TABLE profile_relationship
    ADD COLUMN relationships_type TEXT;

UPDATE profile_relationship
SET relationships_type = CASE relationships_status
    WHEN 'blocked' THEN 'blocked'
    ELSE 'friendship'
END;

CREATE TYPE relationships_status_enum AS ENUM (
    'request',
    'accepted',
    'canceled'
);

ALTER TABLE profile_relationship
    ALTER COLUMN relationships_status TYPE relationships_status_enum
    USING CASE relationships_status
        WHEN 'friend_request' THEN 'request'::relationships_status_enum
        WHEN 'blocked' THEN 'accepted'::relationships_status_enum
        WHEN 'friendship' THEN 'accepted'::relationships_status_enum
        ELSE 'canceled'::relationships_status_enum
    END;

ALTER TABLE profile_relationship
    ALTER COLUMN relationships_type SET NOT NULL,
    ADD CONSTRAINT profile_relationship_type_length
        CHECK (LENGTH(relationships_type) BETWEEN 1 AND 32);

DROP TYPE relationships_enum;

-- Между двумя профилями может существовать только одна активная связь
CREATE UNIQUE INDEX profile_relationship_active_pair_unique
    ON profile_relationship (
        LEAST(profile_id_master, profile_id_slave),
        GREATEST(profile_id_master, profile_id_slave)
    )
    WHERE deleted_at IS NULL
      AND relationships_status <> 'canceled';
