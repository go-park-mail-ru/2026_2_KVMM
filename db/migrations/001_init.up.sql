CREATE TYPE role_enum AS ENUM (
    'admin',
    'moderator',
    'owner',
    'member'
);

CREATE TYPE relationships_enum AS ENUM (
    'friendship',
    'blocked',
    'friend_request'
);

CREATE TYPE gender_enum AS ENUM (
    'male',
    'female',
    'other'
);


-- Пользователи
CREATE TABLE profile (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    nickname TEXT NOT NULL
        CHECK (LENGTH(nickname) BETWEEN 4 AND 32),

    email TEXT NULL,
    phone_number TEXT NULL,

    profile_name TEXT NOT NULL
        CHECK (LENGTH(profile_name) BETWEEN 2 AND 32),
    surname TEXT NOT NULL
        CHECK (LENGTH(surname) BETWEEN 2 AND 32),
    patronymic TEXT NULL
        CHECK (LENGTH(patronymic) BETWEEN 2 AND 32),
    gender gender_enum NOT NULL,
    birthday DATE NULL,
    bio TEXT NULL
        CHECK (LENGTH(bio) <= 256),

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    password_hash TEXT NOT NULL
        CHECK (LENGTH(password_hash) <= 512),
    password_salt TEXT NOT NULL,


    PRIMARY KEY (id),

    CHECK 
        (email IS NOT NULL OR phone_number IS NOT NULL),

    CONSTRAINT profile_nickname_unique
        UNIQUE (nickname),
    CONSTRAINT profile_email_unique
        UNIQUE (email),
    CONSTRAINT profile_phone_unique
        UNIQUE (phone_number)

);
CREATE INDEX profile_name ON profile(profile_name);
CREATE INDEX profile_surname ON profile(surname);
CREATE INDEX profile_patronymic ON profile(patronymic);


-- Сообщества пользователей
CREATE TABLE community (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    community_name TEXT NOT NULL
        CHECK (LENGTH(community_name) <= 128),
    bio TEXT NOT NULL
        CHECK (LENGTH(bio) <= 256),

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id),

    CONSTRAINT community_name_unique
        UNIQUE (community_name)
);



-- Публикации пользователей и сообществ
CREATE TABLE post (
    id BIGINT GENERATED ALWAYS AS IDENTITY,

    post_text TEXT NULL
        CHECK (LENGTH(post_text) <= 256),

    author_profile_id BIGINT NULL,
    author_community_id BIGINT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id),

    CONSTRAINT post_author_check
        CHECK (
            (author_profile_id IS NOT NULL
             AND author_community_id IS NULL)
            OR
            (author_profile_id IS NULL
             AND author_community_id IS NOT NULL)
        ),

    CONSTRAINT post_author_profile_fk
        FOREIGN KEY (author_profile_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    CONSTRAINT post_author_community_fk
        FOREIGN KEY (author_community_id)
        REFERENCES community (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Медиафайлы
CREATE TABLE media (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    media_path TEXT NOT NULL, -- S3 path

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id)
);


-- Медиафайлы публикаций
CREATE TABLE post_media (
    post_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,

    PRIMARY KEY (post_id, media_id),

    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (media_id)
        REFERENCES media (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Репосты публикаций
CREATE TABLE profile_repost (
    profile_id BIGINT NOT NULL,
    post_id BIGINT NOT NULL,

    repost_comment TEXT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (profile_id, post_id),

    FOREIGN KEY (profile_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Лайки публикаций
CREATE TABLE profile_like (
    profile_id BIGINT NOT NULL,
    post_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (profile_id, post_id),

    FOREIGN KEY (profile_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Чаты
CREATE TABLE chat (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id)
);


-- Участники чатов
CREATE TABLE chat_profile (
    profile_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,

    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (profile_id, chat_id),

    FOREIGN KEY (profile_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (chat_id)
        REFERENCES chat (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Сообщения в чатах
CREATE TABLE message (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    chat_id BIGINT NOT NULL,

    message_text TEXT NULL
        CHECK (LENGTH(message_text) <= 256),

    author_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,

    repost_from BIGINT NULL,
    reply_from BIGINT NULL,

    PRIMARY KEY (id),

    FOREIGN KEY (author_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (chat_id)
        REFERENCES chat (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (repost_from)
        REFERENCES message (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (reply_from)
        REFERENCES message (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Медиафайлы сообщений
CREATE TABLE message_media (
    message_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,

    PRIMARY KEY (message_id, media_id),

    FOREIGN KEY (message_id)
        REFERENCES message (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (media_id)
        REFERENCES media (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Связи между пользователями (дружба, бан и тд)
CREATE TABLE profile_relationship (
    profile_id_master BIGINT NOT NULL,
    profile_id_slave BIGINT NOT NULL,

    status relationships_enum NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (profile_id_master, profile_id_slave),

    FOREIGN KEY (profile_id_master)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (profile_id_slave)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    CHECK (profile_id_master <> profile_id_slave)
);


-- Участники сообществ
CREATE TABLE community_member (
    profile_id BIGINT NOT NULL,

    community_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    role role_enum NOT NULL,

    PRIMARY KEY (profile_id, community_id),

    FOREIGN KEY (profile_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (community_id)
        REFERENCES community (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Эмодзи реакций
CREATE TABLE emoji_reaction (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    emoji_path TEXT NOT NULL, -- S3 path

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id)
);


-- Реакции на сообщения
CREATE TABLE message_like (
    message_id BIGINT NOT NULL,

    like_by BIGINT NOT NULL,

    emoji_id BIGINT NOT NULL DEFAULT 1, -- 1 -> 👍

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (message_id, like_by),

    FOREIGN KEY (message_id)
        REFERENCES message (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (like_by)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (emoji_id)
        REFERENCES emoji_reaction (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Комментарии к публикациям
CREATE TABLE comment (
    id BIGINT GENERATED ALWAYS AS IDENTITY,

    post_id BIGINT NOT NULL,

    comment_text TEXT NULL
        CHECK (LENGTH(comment_text) <= 256),

    author_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,

    repost_from BIGINT NULL,
    reply_from BIGINT NULL,


    PRIMARY KEY (id),

    FOREIGN KEY (author_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (repost_from)
        REFERENCES comment (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (reply_from)
        REFERENCES comment (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Реакции на комментарии
CREATE TABLE comment_like (
    comment_id BIGINT NOT NULL,

    like_by BIGINT NOT NULL,

    emoji_id BIGINT NOT NULL DEFAULT 1, -- 1 -> 👍

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (comment_id, like_by),

    FOREIGN KEY (comment_id)
        REFERENCES comment (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (like_by)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (emoji_id)
        REFERENCES emoji_reaction (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Медиафайлы комментариев
CREATE TABLE comment_media (
    comment_id BIGINT NOT NULL,

    media_id BIGINT NOT NULL,

    PRIMARY KEY (comment_id, media_id),

    FOREIGN KEY (comment_id)
        REFERENCES comment (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (media_id)
        REFERENCES media (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Стикеры
CREATE TABLE sticker (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    sticker_path TEXT NOT NULL, -- S3 path

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id)
);


-- Стикеры сообщений
CREATE TABLE message_sticker (
    message_id BIGINT NOT NULL,
    sticker_id BIGINT NOT NULL,

    PRIMARY KEY (message_id, sticker_id),

    FOREIGN KEY (message_id)
        REFERENCES message (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (sticker_id)
        REFERENCES sticker (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Стикеры комментариев
CREATE TABLE comment_sticker (
    comment_id BIGINT NOT NULL,
    sticker_id BIGINT NOT NULL,

    PRIMARY KEY (comment_id, sticker_id),

    FOREIGN KEY (comment_id)
        REFERENCES comment (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (sticker_id)
        REFERENCES sticker (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


-- Аватары пользователей
CREATE TABLE profile_avatar (
    profile_id BIGINT NOT NULL,

    media_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (profile_id, media_id),

    FOREIGN KEY (profile_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (media_id)
        REFERENCES media (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);
