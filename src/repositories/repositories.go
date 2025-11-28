package repositories

import (
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	Article   ArticleRepository
	UserEvent UserEventRepository
}

// NewRepositories creates and returns all repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Article:   NewArticleRepository(db),
		UserEvent: NewUserEventRepository(db),
	}
}
