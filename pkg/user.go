package pkg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jensilo/harmony/pkg/db"
	"github.com/jensilo/harmony/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidLogin = errors.New("invalid login credentials")
)

type UserService struct {
	queries *db.Queries
}

func NewUserService(queries *db.Queries) *UserService {
	return &UserService{queries: queries}
}

func (us *UserService) Register(ctx context.Context, u types.UserToCreate) (pgtype.UUID, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(u.PlainPassword), bcrypt.DefaultCost)
	if err != nil {
		return pgtype.UUID{}, errors.New("could not hash password")
	}

	id, err := us.queries.CreateUser(ctx, db.CreateUserParams{
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Password:  string(passwordHash),
	})
	if err != nil {
		// todo handle more explicitly
		return pgtype.UUID{}, err
	}

	return id, nil
}

func (us *UserService) Login(ctx context.Context, email, password string) (userID, sessionID string, err error) {
	usr, err := us.queries.UserByEmail(ctx, email)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		err = ErrInvalidLogin
		return
	}
	if err != nil {
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password))
	if err != nil {
		err = ErrInvalidLogin
		return
	}

	expiry := time.Now().Add(time.Hour)
	id, err := us.queries.CreateSession(ctx, db.CreateSessionParams{UserID: usr.ID, ExpiryAt: pgtype.Timestamp{Time: expiry, Valid: true}})
	if err != nil {
		return
	}

	userID = usr.ID.String()
	sessionID = id.String()

	return
}
