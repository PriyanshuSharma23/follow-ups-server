package data

import (
	"database/sql"
	"errors"
)

var (
	ErrNoRecordFound  = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	ErrDuplicateEmail = errors.New("duplicate email")
)

type Models struct {
	Users    UsersModel
	Tokens   TokensModel
	Vehicles VehicleModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		UsersModel{DB: db},
		TokensModel{DB: db},
		VehicleModel{DB: db},
	}
}
