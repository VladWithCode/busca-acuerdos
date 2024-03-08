package db

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id                    string `json:"id"`
	Name                  string `json:"name"`
	Lastname              string `json:"lastname"`
	Username              string `json:"username"`
	Email                 string `json:"email"`
	Phone                 string `json:"phone"`
	Password              string `json:"password"`
	Alerts                []Alert
	SubscriptionActive    bool      `json:"subscriptionActive"`
	SubscriptionExpiresAt time.Time `json:"subscriptionExpiresAt"`
}

func (u *User) ValidatePass(pw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pw))

	if err != nil {
		return err
	}

	return nil
}

func CreateUser(id, name, lastname, username, email, phone, password string, subscriptionActive bool) (string, error) {
	conn, err := GetPool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return "", err
	}

	var user User

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	err = conn.QueryRow(
		ctx,
		"INSERT INTO users (id, name, lastname, password, subscription_active, subscription_expires_at, username, email, phone_number) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
		id,
		name,
		lastname,
		hashedPassword,
		subscriptionActive,
		time.Now(),
		username,
		email,
		phone,
	).Scan(
		&user.Id,
	)

	if err != nil {
		return "", err
	}

	return user.Id, nil
}

func GetUserById(id string) (*User, error) {
	conn, err := GetPool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

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
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

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
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
