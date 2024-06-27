package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
	ScopeRefresh        = "refresh"
	ScopePasswordReset  = "password-reset"
)

type Token struct {
	Plaintext string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return &token, nil
}

func ValidatePlaintextToken(v *validator.Validator, plaintextToken string) {
	v.Check(plaintextToken != "", "token", "must be provided")
	v.Check(len(plaintextToken) == 26, "token", "must be 26 bytes long")
}

type TokensModel struct {
	DB *sql.DB
}

func (m TokensModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

func (m TokensModel) Insert(token *Token) error {
	stmt := `
          INSERT INTO tokens (hash, user_id, expiry, scope)
          VALUES ($1, $2, $3, $4)
          `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}
	_, err := m.DB.ExecContext(ctx, stmt, args...)

	return err
}

func (m TokensModel) DeleteAllForUser(userID int64, scope string) error {
	stmt := `
          DELETE FROM tokens
          WHERE user_id=$1 AND scope=$2
          `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{userID, scope}
	_, err := m.DB.ExecContext(ctx, stmt, args...)

	return err
}

func (m TokensModel) ClearAllForUser(userID int64) error {
	stmt := `
          DELETE FROM tokens
          WHERE user_id=$1
          `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{userID}
	_, err := m.DB.ExecContext(ctx, stmt, args...)

	return err
}
