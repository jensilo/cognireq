package pkg

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists  = errors.New("email already exists")
	ErrInvalidLogin = errors.New("invalid login credentials")
	ErrExpired      = errors.New("expired")
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

func (us *UserService) Login(ctx context.Context, email, password string) (userID, sessionID string, err error) {
	usr, err := us.queries.UserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(
			[]byte("$2a$10$6zBr8NNH1S.2Nnyb4UdG/Op12FmR5VPIQHTSs.gAqD.AXhtOL9IIe"),
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

	err = us.queries.UpdateLastLogin(ctx, usr.ID)
	if err != nil {
		return "", "", err
	}

	return usr.ID.String(), sessID.String(), nil
}

func (us *UserService) createLoginSession(ctx context.Context, usrID uuid.UUID) (uuid.UUID, error) {
	expiry := time.Now().Add(time.Hour)
	return us.queries.CreateSession(
		ctx,
		db.CreateSessionParams{UserID: usrID, ExpiryAt: pgtype.Timestamp{Time: expiry, Valid: true}},
	)
}

func (us *UserService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return us.queries.DeleteSession(ctx, sessionID)
}

func (us *UserService) LoggedInUser(ctx context.Context, sessionID uuid.UUID) (*types.User, error) {
	row, err := us.queries.UserBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	isExpired := row.ExpiryAt.Time.Before(time.Now())
	if isExpired {
		return nil, ErrExpired
	}

	return &types.User{
		ID:        row.ID,
		Email:     row.Email,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		LastLogin: row.LastLogin.Time,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// DeleteExpiredSessions deletes all sessions older than a certain threshold (see SQL)
func (us *UserService) DeleteExpiredSessions(ctx context.Context) (int, error) {
	c, err := us.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return 0, err
	}

	return int(c), nil
}

// StartSessionCleanup starts the background job to delete expired sessions periodically.
func StartSessionCleanup(ctx context.Context, logger *slog.Logger, service *UserService, interval time.Duration) {
	logger = logger.With("job", "SessionCleanup")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("started session cleanup routine", "interval", interval.String())
	rows, err := service.DeleteExpiredSessions(ctx)
	if err != nil {
		logger.Error("initial session cleanup failed", "err", err)
	}
	logger.Info("cleaned up initially", "rows affected", rows)

	for {
		select {
		case <-ctx.Done():
			logger.Info("session cleanup routine shutting down")
			return
		case <-ticker.C:
			rows, err := service.DeleteExpiredSessions(ctx)
			if err != nil {
				logger.Error("session cleanup failed", "err", err)
			}
			logger.Info("cleaned up", "rows affected", rows)
		}
	}
}
