# Описание таблиц

## Пользователи
**profile** - Пользователи\
**profile_avatar** - Аватары пользователей\
**profile_repost** - Репосты публикаций\
**profile_like** - Лайки публикаций\
**profile_relationship** - Связи между пользователями

## Сообщества
**community** - Сообщества пользователей\
**community_member** - Участники сообществ

## Публикации
**post** - Публикации пользователей и сообществ\
**post_media** - Медиафайлы публикаций\
**comment** - Комментарии к публикациям\
**comment_like** - Реакции на комментарии\
**comment_media** - Медиафайлы комментариев\
**comment_sticker** - Стикеры комментариев

## Чаты
**chat** - Чаты\
**chat_profile** - Участники чатов\
**message** - Сообщения в чатах\
**message_media** - Медиафайлы сообщений\
**message_like** - Реакции на сообщения\
**message_sticker** - Стикеры сообщений

## Пути к медиафайлам в S3
**media** - Медиафайлы\
**sticker** - Стикеры\
**emoji_reaction** - Эмодзи реакций

# Функциональные зависимости

### profile
```
{ id } -> { nickname, email, phone_number, profile_name, surname, patronymic,gender, birthday, created_at, deleted_at, updated_at, password_hash, password_salt, bio }

{ nickname } -> { id, email, phone_number, profile_name, surname, patronymic,
gender, birthday, created_at, deleted_at, updated_at,
password_hash, password_salt, bio }
```
### community
```
{ id } -> { community_name, bio, created_at, deleted_at, updated_at }
{ community_name } -> { id, bio, created_at, deleted_at, updated_at }
```
### post
```
{ id } -> { post_text, author_profile_id, author_community_id, created_at, updated_at, deleted_at }
```
### media
```
{ media_id } -> { media_path, created_at, updated_at, deleted_at }
```
### post_media
```
{ post_id, media_id } -> { }
```
### profile_repost
```
{ profile_id, post_id } -> { repost_comment, created_at, updated_at, deleted_at }
```
### profile_like
```
{ profile_id, post_id } -> { created_at, deleted_at }
```
### chat
```
{ id } -> { created_at }
```
### chat_profile
```
{ profile_id, chat_id } -> { deleted_at }
```
### message
```
{ id } -> { chat_id, message_text, author_id, created_at, deleted_at,
updated_at, repost_from, reply_from }
```
### message_media
```
{ message_id, media_id } -> { }
```
### profile_relationship
```
{ profile_id_master, profile_id_slave } -> { status, created_at, updated_at, deleted_at }
```
### community_member
```
{ profile_id, community_id } -> { role, created_at, updated_at, deleted_at }
```
### emoji_reaction
```
{ id } -> { emoji_path, created_at, updated_at, deleted_at }
```
### message_like
```
{ message_id, like_by } -> { emoji_id, created_at, updated_at, deleted_at }
```
### comment
```
{ id } -> { post_id, comment_text, author_id, created_at,
deleted_at, updated_at, repost_from, reply_from }
```
### comment_like
```
{ comment_id, like_by } -> { emoji_id, created_at, updated_at, deleted_at }
```
### comment_media
```
{ comment_id, media_id } -> { }
```
### sticker
```
{ id } -> { sticker_path, created_at, updated_at, deleted_at }
```
### message_sticker
```
{ message_id, sticker_id } -> { }
```
### comment_sticker
```
{ comment_id, sticker_id } -> { }
```
### profile_avatar
```
{ profile_id, media_id } -> { created_at, deleted_at }
```
# Соответствие первой нормальной форме (1НФ)

Переменная отношения находится в первой нормальной форме (1НФ) тогда и только тогда, когда в любом допустимом значении отношения каждый его кортеж содержит только одно значение для каждого из атрибутов.

В базе каждое используемое значеине имеет свое поле (атомарность значений) например:
```
profile.nickname
profile.email
profile.birthday
post.post_text
message.message_text
community.community_name
...
```

Отсуствие списков в полях реализуется через промежуточные таблицы например вместо хранения списка медиа файлов в post используется таблица post_media

Таким образом, все отношения находятся в 1НФ.

# Соответствие второй нормальной форме (2НФ)

Переменная отношения находится во второй нормальной форме тогда и только тогда, когда она находится в первой нормальной форме, и каждый неключевой атрибут неприводимо (функционально полно) зависит от ее потенциального ключа.


Например:
```
profile:

{ id } -> { nickname, email, profile_name, surname, ... }
```
id состоит из одного атрибута.

То же самое:
```
post:

{ id } -> { post_text, author_profile_id, ... }
```

Для отношений с составными ключами зависимости также являются зависимостями от всего ключа.

Например:
```
profile_repost:

{ profile_id, post_id } -> { repost_comment }
```

repost_comment зависит от комбинации пользователя и поста, а не только от:
```
profile_id
```

или только:
```
post_id
```

Следовательно, частичных функциональных зависимостей не возникает, и отношения находятся в 2НФ.

# Соответствие третьей нормальной форме (3НФ)

Переменная отношения находится в третьей нормальной форме, когда она находится во второй нормальной форме, и отсутствуют транзитивные функциональные зависимости неключевых атрибутов от ключевых. (любой столбец таблицы должен зависеть только от ключевого столбца.)

В  атрибуты сущностей вынесены в соответствующие отношения.

Например, в message:
```
{ id } -> { chat_id, author_id, message_text, created_at, ... }
```

Данные чата и автора не хранятся в message.

Информация о чате находится отдельно находится в:
```
chat:
{ id } -> { created_at }
```
Информация о пользователе отдельно находится в:
```
profile:
{ id } -> { nickname, profile_name, surname, ... }
```

Таким образом, например, имя автора сообщения не хранится в message.Это предотвращает зависимость вида:
```
message_id -> author_id -> author_name
```

Следовательно, транзитивные зависимости между неключевыми атрибутами в отношениях отсутствуют, поэтому отношения находятся в 3НФ.

# Соответствие НФБК (BCNF)

Переменная отношения находится в BCNF тогда и только тогда, когда каждая её нетривиальная и неприводимая слева функциональная зависимость имеет в качестве своего детерминанта некоторый потенциальный ключ.

Пример: отношение profile
Функциональные зависимости:
```
{ id } -> { nickname, email, profile_name, surname... }
```
id является первичным ключом, поэтому он является потенциальным ключом.
Также:
```
{ nickname } -> { id, email, profile_name, surname.. }
```
nickname имеет ограничение UNIQUE, поэтому он также является потенциальным ключом.
Обе функциональные зависимости являются нетривиальными и неприводимыми слева, а их детерминанты (id и nickname) являются потенциальными ключами. Следовательно, отношение profile соответствует НФБК

Аналогично рассмотрев остальные таблицы видно что, для каждой нетривиальной функциональной зависимости детерминант является суперключом. Следовательно, отношения соответствуют **НФБК**.
