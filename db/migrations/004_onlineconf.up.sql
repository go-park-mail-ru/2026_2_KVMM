-- Конфигурационные значения и feature flags сервиса OnlineConf.
CREATE TABLE onlineconf_entry (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    uid UUID NOT NULL DEFAULT gen_random_uuid(),

    scope TEXT NOT NULL
        CHECK (scope IN ('frontend', 'auth', 'user', 'posts', 'community', 'backend')),
    name TEXT NOT NULL
        CHECK (LENGTH(name) BETWEEN 1 AND 128),
    kind TEXT NOT NULL
        CHECK (kind IN ('config', 'flag')),

    value JSONB NULL,
    rollout SMALLINT NULL,

    created_by_profile_id BIGINT NOT NULL,
    updated_by_profile_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id),
    UNIQUE (uid),

    CHECK (
        (kind = 'config' AND value IS NOT NULL AND rollout IS NULL)
        OR
        (kind = 'flag' AND value IS NULL AND rollout BETWEEN 0 AND 100)
    )
);

-- Имена ключей не должны отличаться только регистром.
CREATE UNIQUE INDEX onlineconf_entry_scope_name_unique
    ON onlineconf_entry (scope, LOWER(name));

-- Принудительное включение или исключение пользователя из раскатки feature flag.
CREATE TABLE onlineconf_flag_target (
    entry_id BIGINT NOT NULL,
    profile_id BIGINT NOT NULL,
    profile_uid UUID NOT NULL,
    mode TEXT NOT NULL
        CHECK (mode IN ('include', 'exclude')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (entry_id, profile_id),
    UNIQUE (entry_id, profile_uid),

    CONSTRAINT onlineconf_flag_target_entry_fk
        FOREIGN KEY (entry_id)
        REFERENCES onlineconf_entry (id)
        ON DELETE CASCADE
);


