package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id                    string         `json:"id" db:"id"`
	Name                  string         `json:"name" db:"name"`
	Lastname              string         `json:"lastname" db:"lastname"`
	Username              string         `json:"username" db:"username"`
	Email                 string         `json:"email" db:"email"`
	Password              string         `json:"password" db:"password"`
	Phone                 sql.NullString `json:"phone" db:"phone_number"`
	SubscriptionActive    bool           `json:"subscriptionActive" db:"subscription_active"`
	SubscriptionExpiresAt time.Time      `json:"subscriptionExpiresAt" db:"subscription_expires_at"`
	EmailVerified         bool           `json:"emailVerified" db:"email_verified"`
	PhoneVerified         bool           `json:"phoneVerified" db:"phone_verified"`
	Alerts                []Alert
}

func (u *User) ValidatePass(pw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pw))

	if err != nil {
		return err
	}

	return nil
}

func (u *User) CreateReportDir() error {
	tsjDir := os.Getenv("TSJ_DIR")
	if tsjDir != "" {
		tsjDir = tsjDir + "/"
	}
	reportsDir := fmt.Sprintf("%vweb/static/reports/%v", tsjDir, u.Id)

	err := os.MkdirAll(reportsDir, fs.ModeDir)

	if err != nil {
		return err
	}

	return nil
}

func CreateUser(user *User) (string, error) {
	conn, err := GetPool()
	if err != nil {
		return "", err
	}
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return "", err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	tag, err := conn.Exec(
		ctx,
		"INSERT INTO users (id, name, lastname, password, subscription_active, subscription_expires_at, username, email, phone_number) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		user.Id,
		user.Name,
		user.Lastname,
		hashedPassword,
		user.SubscriptionActive,
		time.Now(),
		user.Username,
		user.Email,
		user.Phone,
	)

	if err != nil {
		return "", err
	}

	if tag.RowsAffected() == 0 {
		return "", errors.New("No se creó el usuario")
	}

	return user.Id, nil
}

func GetUserById(id string) (*User, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user User

	err = conn.QueryRow(
		ctx,
		"SELECT * FROM users WHERE id = $1",
		id,
	).Scan(
		&user.Id,
		&user.Name,
		&user.Lastname,
		&user.Password,
		&user.SubscriptionActive,
		&user.SubscriptionExpiresAt,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.EmailVerified,
		&user.PhoneVerified,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user User

	err = conn.QueryRow(
		ctx,
		"SELECT * FROM users WHERE username = $1",
		username,
	).Scan(
		&user.Id,
		&user.Name,
		&user.Lastname,
		&user.Password,
		&user.SubscriptionActive,
		&user.SubscriptionExpiresAt,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.EmailVerified,
		&user.PhoneVerified,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func TxVerifyUserEmail(ctx context.Context, tx pgx.Tx, userId string) error {
	tag, err := tx.Exec(
		ctx,
		"UPDATE users SET email_verified = TRUE WHERE id = $1",
		userId,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return errors.New(fmt.Sprintf("No se encontró usuario con id %v", userId))
	}

	return nil
}

func VerifyUserEmail(userId string) error {
	conn, err := GetPool()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tag, err := conn.Exec(
		ctx,
		"UPDATE users SET email_verified = TRUE WHERE id = $1",
		userId,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return errors.New(fmt.Sprintf("No se encontró usuario con id %v", userId))
	}

	return nil
}
