package relationship

type FriendsHandler struct{}

// NewRelationshipHandler создаёт обработчик связей
func NewRelationshipHandler() *FriendsHandler { return &FriendsHandler{} }
