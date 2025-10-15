package types

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	LastName  string
	CreatedAt time.Time
	UpdatedAt time.Time
	LastLogin time.Time
}

type UserToCreate struct {
	Email         string
	FirstName     string
	LastName      string
	PlainPassword string
}
