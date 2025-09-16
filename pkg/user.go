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
	ErrEmailExists  = errors.New("email already exists")
	ErrInvalidLogin = errors.New("invalid login credentials")
)

type UserService struct {
	queries *db.Queries
}

func NewUserService(queries *db.Queries) *UserService {
	return &UserService{queries: queries}
}

func (us *UserService) Register(ctx context.Context, u types.UserToCreate) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(u.PlainPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("could not hash password")
	}

	id, err := us.queries.CreateUser(ctx, db.CreateUserParams{
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Password:  string(passwordHash),
	})
	if db.IsUniqueConstraintViolation(err) {
		return "", ErrEmailExists
	}
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

// todo fix timing attack!
func (us *UserService) Login(ctx context.Context, email, password string) (userID, sessionID string, err error) {
	usr, err := us.queries.UserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(
			[]byte("9a5fe17d0d973036af5bf1e1bd1399bed442bd8dac9ed323a3af04fb347c9ef9"),
			[]byte("6de4f3ca5701c098ff2cf2e45003d0b858f7991064d5c193eb6bfb159e8bcb1e"),
		)
		return "", "", ErrInvalidLogin
	}
	if err != nil {
		return "", "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return "", "", ErrInvalidLogin
	}
	if err != nil {
		return "", "", err
	}

	sessID, err := us.createLoginSession(ctx, usr.ID)
	if err != nil {
		return "", "", err
	}

	return usr.ID.String(), sessID.String(), nil
}

func (us *UserService) createLoginSession(ctx context.Context, usrID pgtype.UUID) (pgtype.UUID, error) {
	expiry := time.Now().Add(time.Hour)
	return us.queries.CreateSession(
		ctx,
		db.CreateSessionParams{UserID: usrID, ExpiryAt: pgtype.Timestamp{Time: expiry, Valid: true}},
	)
}
