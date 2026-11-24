// Package search содержит только Swagger-описания будущих поисковых ручек.
// Функции в этом файле не являются HTTP-обработчиками и не подключены к маршрутизатору.
package search

// SearchPostsContract описывает контракт поиска постов.
// @Summary Поиск публикаций
// @Description Ищет по тексту постов пользователей и сообществ. Страница сортируется по релевантности.
// @Tags search
// @Produce json
// @Param q query string true "Поисковая строка, от 2 до 128 символов" minlength(2) maxlength(128)
// @Param cursor query string false "Непрозрачный курсор из next_cursor предыдущего ответа"
// @Param limit query int false "Размер страницы от 1 до 50" default(20) minimum(1) maximum(50)
// @Success 200 {object} domain.SearchPostsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/posts/search [get]
func SearchPostsContract() {}

// SearchProfilesContract описывает контракт полнотекстового поиска пользователей.
// @Summary Поиск пользователей
// @Description Ищет по nickname, profile_name и surname. Страница сортируется по релевантности.
// @Tags search
// @Produce json
// @Param q query string true "Поисковая строка, от 2 до 128 символов" minlength(2) maxlength(128)
// @Param cursor query string false "Непрозрачный курсор из next_cursor предыдущего ответа"
// @Param limit query int false "Размер страницы от 1 до 50" default(20) minimum(1) maximum(50)
// @Success 200 {object} domain.SearchProfilesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/profiles/search [get]
func SearchProfilesContract() {}

// SearchCommunitiesContract описывает контракт поиска сообществ.
// @Summary Поиск сообществ
// @Description Ищет по названию и описанию сообщества. Страница сортируется по релевантности.
// @Tags search
// @Produce json
// @Param q query string true "Поисковая строка, от 2 до 128 символов" minlength(2) maxlength(128)
// @Param cursor query string false "Непрозрачный курсор из next_cursor предыдущего ответа"
// @Param limit query int false "Размер страницы от 1 до 50" default(20) minimum(1) maximum(50)
// @Success 200 {object} domain.SearchCommunitiesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/communities/search [get]
func SearchCommunitiesContract() {}

// SearchMessagesContract описывает контракт поиска сообщений в доступных пользователю чатах.
// @Summary Поиск сообщений
// @Description Ищет только в чатах, участником которых является текущий пользователь.
// @Tags search
// @Security SessionCookie
// @Produce json
// @Param q query string true "Поисковая строка, от 2 до 128 символов" minlength(2) maxlength(128)
// @Param cursor query string false "Непрозрачный курсор из next_cursor предыдущего ответа"
// @Param limit query int false "Размер страницы от 1 до 50" default(20) minimum(1) maximum(50)
// @Success 200 {object} domain.SearchMessagesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/messages/search [get]
func SearchMessagesContract() {}
