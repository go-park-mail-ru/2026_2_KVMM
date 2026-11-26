package search

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/mozillazg/go-unidecode"
)

type Handler struct{ client *elasticsearch.Client }

const BackfillPageSize = 500

func NewHandler(client *elasticsearch.Client) *Handler { return &Handler{client: client} }
func params(r *http.Request) (string, int, int, error) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if n := len([]rune(q)); n < 2 || n > 128 {
		return "", 0, 0, errors.New("q must contain 2 to 128 characters")
	}
	from := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		b, e := base64.RawURLEncoding.DecodeString(c)
		if e != nil {
			return "", 0, 0, errors.New("invalid cursor")
		}
		from, e = strconv.Atoi(string(b))
		if e != nil || from < 0 || from > 10000 {
			return "", 0, 0, errors.New("invalid cursor")
		}
	}
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > 50 {
			return "", 0, 0, errors.New("limit must be from 1 to 50")
		}
		limit = n
	}
	return q, from, limit, nil
}

// SearchProfiles godoc
// @Summary Поиск пользователей
// @Description Ищет по nickname, profile_name и surname. Страница сортируется по релевантности
// @Tags search
// @Produce json
// @Param q query string true "Поисковая строка, от 2 до 128 символов" minlength(2) maxlength(128)
// @Param cursor query string false "Непрозрачный курсор из next_cursor предыдущего ответа"
// @Param limit query int false "Размер страницы от 1 до 50" default(20) minimum(1) maximum(50)
// @Success 200 {object} domain.SearchProfilesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/profiles/search [get]
func (h *Handler) Profiles(w http.ResponseWriter, r *http.Request) {
	q, from, l, e := params(r)
	if e != nil {
		helpers.WriteError(w, 400, "validation_error", e.Error())
		return
	}
	docs, next, more, e := h.search(r.Context(), "profiles", q, from, l, nil, []string{"text^2", "translit"})
	if e != nil {
		helpers.WriteInternalError(w, "search.profiles", e)
		return
	}
	out := domain.SearchProfilesResponse{Query: q, Profiles: []domain.ShortProfile{}, Limit: l, HasMore: more, NextCursor: next}
	for _, d := range docs {
		var v struct {
			Profile domain.ShortProfile `json:"profile"`
		}
		if e = json.Unmarshal(d, &v); e != nil {
			helpers.WriteInternalError(w, "search.profiles", e)
			return
		}
		out.Profiles = append(out.Profiles, v.Profile)
	}
	helpers.WriteJSON(w, 200, out)
}

// SearchCommunities godoc
// @Summary Поиск сообществ
// @Description Ищет по названию сообщества. Страница сортируется по релевантности
// @Tags search
// @Produce json
// @Param q query string true "Поисковая строка, от 2 до 128 символов" minlength(2) maxlength(128)
// @Param cursor query string false "Непрозрачный курсор из next_cursor предыдущего ответа"
// @Param limit query int false "Размер страницы от 1 до 50" default(20) minimum(1) maximum(50)
// @Success 200 {object} domain.SearchCommunitiesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/communities/search [get]
func (h *Handler) Communities(w http.ResponseWriter, r *http.Request) {
	q, from, l, e := params(r)
	if e != nil {
		helpers.WriteError(w, 400, "validation_error", e.Error())
		return
	}
	docs, next, more, e := h.search(r.Context(), "communities", q, from, l, nil, []string{"text^2", "translit"})
	if e != nil {
		helpers.WriteInternalError(w, "search.communities", e)
		return
	}
	out := domain.SearchCommunitiesResponse{Query: q, Communities: []domain.Community{}, Limit: l, HasMore: more, NextCursor: next}
	for _, d := range docs {
		var v struct {
			Community domain.Community `json:"community"`
		}
		if e = json.Unmarshal(d, &v); e != nil {
			helpers.WriteInternalError(w, "search.communities", e)
			return
		}
		out.Communities = append(out.Communities, v.Community)
	}
	helpers.WriteJSON(w, 200, out)
}
func (h *Handler) search(ctx context.Context, index, q string, from, limit int, filter map[string]any, fields []string) ([]json.RawMessage, string, bool, error) {
	should := []any{map[string]any{"multi_match": map[string]any{"query": q, "fields": fields, "fuzziness": "AUTO"}}}
	if index == "profiles" || index == "communities" {
		should = append(should, map[string]any{"match": map[string]any{"translit": unidecode.Unidecode(q)}})
	}
	bq := map[string]any{"should": should, "minimum_should_match": 1}
	if filter != nil {
		bq["filter"] = []any{map[string]any{"terms": filter}}
	}
	body, e := json.Marshal(map[string]any{"from": from, "size": limit + 1, "query": map[string]any{"bool": bq}, "sort": []any{map[string]any{"_score": "desc"}, map[string]any{"uid.keyword": "asc"}}})
	if e != nil {
		return nil, "", false, e
	}
	res, e := h.client.Search(h.client.Search.WithContext(ctx), h.client.Search.WithIndex(index), h.client.Search.WithBody(bytes.NewReader(body)))
	if e != nil {
		return nil, "", false, e
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return []json.RawMessage{}, "", false, nil
	}
	if res.IsError() {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, "", false, fmt.Errorf("elasticsearch %s: %s", index, strings.TrimSpace(string(b)))
	}
	var result struct {
		Hits struct {
			Hits []struct {
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if e = json.NewDecoder(res.Body).Decode(&result); e != nil {
		return nil, "", false, e
	}
	more := len(result.Hits.Hits) > limit
	if more {
		result.Hits.Hits = result.Hits.Hits[:limit]
	}
	docs := make([]json.RawMessage, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		docs = append(docs, hit.Source)
	}
	next := ""
	if more {
		next = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(from + limit)))
	}
	return docs, next, more, nil
}

// IndexProfile сохраняет в индексе только публичные данные профиля
func (h *Handler) IndexProfile(ctx context.Context, p domain.Profile) error {
	short := domain.ShortProfile{UID: p.UID, Nickname: p.Nickname, ProfileName: p.ProfileName, Surname: p.Surname, Patronymic: p.Patronymic}
	text := strings.TrimSpace(short.Nickname + " " + str(short.ProfileName) + " " + str(short.Surname))
	return h.indexDocument(ctx, "profiles", p.UID, map[string]any{"uid": p.UID, "profile": short, "text": text, "translit": unidecode.Unidecode(text)})
}

// IndexCommunity сохраняет сообщество в индексе Elasticsearch
func (h *Handler) IndexCommunity(ctx context.Context, p domain.Community) error {
	text := strings.TrimSpace(p.Name)
	return h.indexDocument(ctx, "communities", p.UID, map[string]any{"uid": p.UID, "community": p, "text": text, "translit": unidecode.Unidecode(text)})
}

// ResetIndex очищает индекс перед постраничной пересборкой
func (h *Handler) ResetIndex(ctx context.Context, index string) error {
	return h.resetIndex(ctx, index)
}

// IndexProfiles индексирует страницу профилей
func (h *Handler) IndexProfiles(ctx context.Context, profiles []domain.Profile) error {
	for _, profile := range profiles {
		if err := h.IndexProfile(ctx, profile); err != nil {
			return fmt.Errorf("index profile %s: %w", profile.UID, err)
		}
	}
	return nil
}

// IndexCommunities индексирует страницу сообществ
func (h *Handler) IndexCommunities(ctx context.Context, communities []domain.Community) error {
	for _, community := range communities {
		if err := h.IndexCommunity(ctx, community); err != nil {
			return fmt.Errorf("index community %s: %w", community.UID, err)
		}
	}
	return nil
}

func (h *Handler) resetIndex(ctx context.Context, index string) error {
	res, err := h.client.Indices.Delete([]string{index}, h.client.Indices.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.IsError() {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("elasticsearch reset %s: %s", index, strings.TrimSpace(string(body)))
	}
	return nil
}

// DeleteDocument удаляет сущность из указанного поискового индекса
func (h *Handler) DeleteDocument(ctx context.Context, index, uid string) error {
	res, err := h.client.Delete(index, uid, h.client.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return nil
	}
	if res.IsError() {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("elasticsearch delete %s: %s", index, string(b))
	}
	return nil
}
func (h *Handler) indexDocument(ctx context.Context, index, uid string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := h.client.Index(index, bytes.NewReader(body), h.client.Index.WithContext(ctx), h.client.Index.WithDocumentID(uid), h.client.Index.WithRefresh("wait_for"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("elasticsearch index %s: %s", index, string(b))
	}
	return nil
}
func str(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
