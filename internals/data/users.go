package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/validator"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	ID        int64     `json:"id"`
	Version   int       `json:"-"`
	Activated bool      `json:"activated"`
}

var AnonymousUser = new(User)

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.hash = hash
	p.plaintext = &plaintextPassword

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func ValidatePasswordPlaintext(v *validator.Validator, plaintextPassword string) {
	v.Check(validator.NotBlank(plaintextPassword), "password", "must be provided")
	v.Check(len(plaintextPassword) >= 8, "password", "must be atleast 8 bytes long.")
	v.Check(len(plaintextPassword) <= 72, "password", "must be atmost 72 bytes long.")
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(validator.NotBlank(email), "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRx), "email", "must be a valid email address")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	// Remeber to Set password before calling this function
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UsersModel struct {
	DB *sql.DB
}

func (m UsersModel) Insert(user *User) error {
	stmt := `INSERT INTO users (name, email, password_hash, activated)
		     VALUES ($1, $2, $3, $4) 
			 RETURNING id, created_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{user.Name, user.Email, user.Password.hash, user.Activated}
	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "users_email_key"):
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UsersModel) GetByEmail(email string) (*User, error) {
	stmt := `SELECT id, created_at, name, email, password_hash, activated, version 
	 		 FROM users
			 WHERE email=$1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User

	err := m.DB.QueryRowContext(ctx, stmt, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecordFound
		}

		return nil, err
	}

	return &user, nil
}

func (m UsersModel) GetForToken(tokenPlaintext, scope string) (*User, error) {
	stmt := `
          SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.activated, users.version 
          FROM users
          INNER JOIN tokens
          ON users.id = tokens.user_id
          WHERE tokens.hash = $1 AND tokens.scope = $2 AND tokens.expiry > $3`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	args := []any{tokenHash[:], scope, time.Now()}

	var user User
	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecordFound
		}

		return nil, err
	}

	return &user, nil
}

func (m UsersModel) UpdateUser(user *User) error {
	stmt := `
			UPDATE users 
			SET name=$1, email=$2, password_hash=$3, activated=$4, version=version + 1
			WHERE id=$5 AND version=$6
			RETURNING version
			`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "users_email_key"):
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}
