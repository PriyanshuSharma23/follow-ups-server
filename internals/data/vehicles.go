package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/validator"
)

type Vehicle struct {
	ID           int64     `json:"id"`
	LicensePlate string    `json:"license_plate"`
	Make         string    `json:"make"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
	Vin          string    `json:"vin"`
	Color        string    `json:"color"`
	BodyType     string    `json:"body_type"`
	CreatedAt    time.Time `json:"created_at"`
	Version      int       `json:"version"`
}

func ValidateVehicle(v *validator.Validator, vehicle *Vehicle) {
	v.Check(validator.NotBlank(vehicle.LicensePlate), "license_plate", "license_plate must not be blank")

	v.Check(validator.NotBlank(vehicle.Make), "make", "make must not be blank")

	v.Check(validator.NotBlank(vehicle.Model), "model", "model must not be blank")

	v.Check(vehicle.Year > 1885, "year", "year must be more than or equal to 1885")
	currYear := time.Now().Year()
	v.Check(vehicle.Year <= currYear, "year", fmt.Sprintf("year must be less than or equal to %d", currYear))

	v.Check(validator.NotBlank(vehicle.Model), "vin", "vin must not be blank")
	v.Check(validator.NotBlank(vehicle.Color), "color", "color must not be blank")
	v.Check(validator.NotBlank(vehicle.Model), "body_type", "body_type must not be blank")
}

type VehicleModel struct {
	DB *sql.DB
}

func (m VehicleModel) Insert(vehicle *Vehicle) error {
	stmt := `INSERT INTO vehicles (license_plate, make, model, year, vin, color, body_type)
          VALUES ($1, $2, $3, $4, $5, $6, $7)
          RETURNING id, created_at, version`

	args := []any{vehicle.LicensePlate, vehicle.Make, vehicle.Model, vehicle.Year, vehicle.Vin, vehicle.Color, vehicle.BodyType}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	return m.DB.QueryRowContext(ctx, stmt, args...).Scan(&vehicle.ID, &vehicle.CreatedAt, &vehicle.Version)
}

func (m VehicleModel) Get(id int) (*Vehicle, error) {
	if id < 0 {
		return nil, ErrNoRecordFound
	}

	var vehicle Vehicle

	stmt := `SELECT id, license_plate, make, model, year, vin, color, body_type, created_at, version
           FROM vehicles
           WHERE id=$1`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, id).Scan(
		vehicle.ID,
		vehicle.LicensePlate,
		vehicle.Make,
		vehicle.Model,
		vehicle.Year,
		vehicle.Vin,
		vehicle.Color,
		vehicle.BodyType,
		vehicle.CreatedAt,
		vehicle.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecordFound
		}
		return nil, err
	}

	return &vehicle, nil
}

func (m VehicleModel) Update(vehicle *Vehicle) error {
	stmt := `UPDATE vehicles
           SET license_plate=$1, make=$2, model=$3, year=$4, vin=$5, color=$6, body_type=$7, version = version + 1
           WHERE id=$9 AND version=$10
           RETURNING version;`

	args := []any{vehicle.LicensePlate, vehicle.Make, vehicle.Model, vehicle.Year, vehicle.Vin, vehicle.Color, vehicle.BodyType, vehicle.ID, vehicle.Version}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(&vehicle.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m VehicleModel) Delete(id int) error {
	stmt := `DELETE FROM vehicles
           WHERE id=$1`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	r, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNoRecordFound
	}

	return nil
}

func (m VehicleModel) GetAll(f Filters) ([]*Vehicle, Metadata, error) {
	stmt := fmt.Sprintf(`SELECT COUNT(*) OVER(), id, license_plate, make, model, year, vin, color, body_type, created_at, version
           FROM vehicles
		   ORDER BY %s %s, id ASC
		   LIMIT $1 OFFSET $2`, f.sortColumn(), f.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{
		f.limit(),
		f.offset(),
	}

	rows, err := m.DB.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	var totalRecords int

	vehicles := make([]*Vehicle, 0)

	for rows.Next() {
		var c Vehicle

		err := rows.Scan(
			&totalRecords,
			&c.ID,
			&c.LicensePlate,
			&c.Make,
			&c.Model,
			&c.Year,
			&c.Vin,
			&c.Color,
			&c.BodyType,
			&c.CreatedAt,
			&c.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		vehicles = append(vehicles, &c)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, f.Page, f.PageSize)

	return vehicles, metadata, nil
}
