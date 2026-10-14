package posts

type PostHandler struct{}

// NewProfileHandler создаёт обработчик профиля
func NewPostHandler() *PostHandler { return &PostHandler{} }
