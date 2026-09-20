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


CREATE TABLE profile (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    nickname TEXT NOT NULL
        CHECK (LENGTH(nickname) BETWEEN 4 AND 32),

    email TEXT NULL,
    phone_number TEXT NULL,

    name TEXT NOT NULL
        CHECK (LENGTH(name) BETWEEN 2 AND 32),
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

    CONSTRAINT user_nickname_unique
        UNIQUE (nickname),
    CONSTRAINT user_email_unique
        UNIQUE (email),
    CONSTRAINT user_phone_unique
        UNIQUE (phone_number)

);
CREATE INDEX user_name ON profile(name);
CREATE INDEX user_surname ON profile(surname);
CREATE INDEX user_patronymic ON profile(patronymic);

CREATE TABLE community (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL
        CHECK (LENGTH(name) <= 128),
    bio TEXT NOT NULL
        CHECK (LENGTH(bio) <= 256),

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id),

    CONSTRAINT community_name_unique
        UNIQUE (name)
);


CREATE TABLE post (
    id BIGINT GENERATED ALWAYS AS IDENTITY,

    post_text TEXT NULL
        CHECK (LENGTH(post_text) <= 256),

    author_user_id BIGINT NULL,
    author_community_id BIGINT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id),

    CONSTRAINT post_author_check
        CHECK (
            (author_user_id IS NOT NULL
             AND author_community_id IS NULL)
            OR
            (author_user_id IS NULL
             AND author_community_id IS NOT NULL)
        ),

    CONSTRAINT post_author_user_fk
        FOREIGN KEY (author_user_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    CONSTRAINT post_author_community_fk
        FOREIGN KEY (author_community_id)
        REFERENCES community (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


CREATE TABLE media (
    media_id BIGINT NOT NULL,
    media_path TEXT NOT NULL, -- S3 path

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (media_id)
);


CREATE TABLE post_media (
    post_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,

    PRIMARY KEY (post_id, media_id),

    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (media_id)
        REFERENCES media (media_id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


CREATE TABLE user_repost (
    user_id BIGINT NOT NULL,
    post_id BIGINT NOT NULL,

    repost_comment TEXT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (user_id, post_id),

    FOREIGN KEY (user_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


CREATE TABLE user_like (
    user_id BIGINT NOT NULL,
    post_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (user_id, post_id),

    FOREIGN KEY (user_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (post_id)
        REFERENCES post (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


CREATE TABLE chat (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id)
);


CREATE TABLE chat_user (
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,

    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (user_id, chat_id),

    FOREIGN KEY (user_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (chat_id)
        REFERENCES chat (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


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


CREATE TABLE message_media (
    message_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (message_id, media_id),

    FOREIGN KEY (message_id)
        REFERENCES message (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (media_id)
        REFERENCES media (media_id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


CREATE TABLE user_relationship (
    user_id_master BIGINT NOT NULL,
    user_id_slave BIGINT NOT NULL,

    status relationships_enum NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (user_id_master, user_id_slave),

    FOREIGN KEY (user_id_master)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (user_id_slave)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    CHECK (user_id_master <> user_id_slave)
);


CREATE TABLE community_member (
    user_id BIGINT NOT NULL,

    community_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    role role_enum NOT NULL,

    PRIMARY KEY (user_id, community_id),

    FOREIGN KEY (user_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (community_id)
        REFERENCES community (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE TABLE emoji_reaction (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    emoji_path TEXT NOT NULL, -- S3 path

    PRIMARY KEY (id)
);


CREATE TABLE message_like (
    message_id BIGINT NOT NULL,

    like_by BIGINT NOT NULL,

    emoji_id BIGINT NOT NULL DEFAULT 0, -- 0 -> 👍

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


CREATE TABLE comment_like (
    comment_id BIGINT NOT NULL,

    like_by BIGINT NOT NULL,

    emoji_id BIGINT NOT NULL DEFAULT 0, -- 0 -> 👍

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


CREATE TABLE comment_media (
    comment_id BIGINT NOT NULL,

    media_id BIGINT NOT NULL,

    PRIMARY KEY (comment_id, media_id),

    FOREIGN KEY (comment_id)
        REFERENCES comment (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    FOREIGN KEY (media_id)
        REFERENCES media (media_id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);


CREATE TABLE sticker (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    sticker_path TEXT NOT NULL, -- S3 path

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (id)
);


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


CREATE TABLE user_avatar (
    user_id BIGINT NOT NULL,

    media_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    PRIMARY KEY (user_id, media_id),

    FOREIGN KEY (user_id)
        REFERENCES profile (id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    FOREIGN KEY (media_id)
        REFERENCES media (media_id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);
