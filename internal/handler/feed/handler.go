package feed

type FeedHandler struct{}

// NewFeedHandler создаёт обработчик запросов ленты
func NewFeedHandler() *FeedHandler { return &FeedHandler{} }
