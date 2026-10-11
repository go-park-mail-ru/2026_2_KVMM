package friends

type FriendsHandler struct{}

// NewFriendsHandler создаёт обработчик запросов ленты
func NewFriendsHandler() *FriendsHandler { return &FriendsHandler{} }
