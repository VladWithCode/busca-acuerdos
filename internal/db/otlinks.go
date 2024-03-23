package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OTLink struct {
	Id        uuid.UUID `json:"id" db:"id"`
	Code      uuid.UUID `json:"code" db:"code"`
	Used      bool      `json:"used" db:"used"`
	UserId    uuid.UUID `json:"userId" db:"user_id"`
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	Action    string    `json:"action" db:"action"`
}

func (otl *OTLink) CheckExpiration() bool {
	return otl.ExpiresAt.Before(time.Now())
}

type NonExistentOTLError struct {
	Err string
}

func (e *NonExistentOTLError) Error() string {
	return e.Err
}

const (
	OTLActionVerify    = "XXVERIFY"
	OTLActionLogin     = "XXXLOGIN"
	OTLActionResetPass = "RESTPASS"
)

func CreateOTLink(userId uuid.UUID, action string) (*OTLink, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	otlId, err := uuid.NewV7()

	if err != nil {
		return nil, err
	}

	otlCode, err := uuid.NewV7()

	if err != nil {
		return nil, err
	}

	otLink := &OTLink{
		Id:        otlId,
		Code:      otlCode,
		Action:    action,
		UserId:    userId,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	tag, err := conn.Exec(
		ctx,
		"INSERT INTO otlinks (id, code, user_id, expires_at, action) VALUES ($1, $2, $3, $4, $5)",
		otLink.Id,
		otLink.Code,
		otLink.UserId,
		otLink.ExpiresAt,
		otLink.Action,
	)

	if err != nil {
		return nil, err
	}

	if tag.RowsAffected() == 0 {
		return nil, errors.New("No se creó el link")
	}

	return otLink, nil
}

func CreateVerifyOTL(userId uuid.UUID) (*OTLink, error) {
	return CreateOTLink(userId, OTLActionVerify)
}

func CreateLoginOTL(userId uuid.UUID) (*OTLink, error) {
	return CreateOTLink(userId, OTLActionLogin)
}

func CreateResetPassOTL(userId uuid.UUID) (*OTLink, error) {
	return CreateOTLink(userId, OTLActionResetPass)
}

func MarkOTLinkAsUsed(code uuid.UUID, userId uuid.UUID) error {
	conn, err := GetPool()
	if err != nil {
		return err
	}
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tag, err := conn.Exec(
		ctx,
		"UPDATE otlinks SET used = TRUE WHERE code = $1 AND user_id = $2",
		code,
		userId,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return &NonExistentOTLError{
			Err: fmt.Sprintf("OTLink con codigo %v para usuario %v no existe", code.String(), userId.String()),
		}
	}

	return nil
}
