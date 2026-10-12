package chat

// ChatHandler обрабатывает запросы к чатам и сообщениям
type ChatHandler struct{}

// NewChatHandler создаёт обработчик чатов
func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}
