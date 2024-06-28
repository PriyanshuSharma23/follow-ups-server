package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/data"
	"github.com/PriyanshuSharma23/follow-ups-server/internals/validator"
)

func (app *application) createVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		LicensePlate string `json:"license_plate"`
		Make         string `json:"make"`
		Model        string `json:"model"`
		Year         int    `json:"year"`
		Vin          string `json:"vin"`
		Color        string `json:"color"`
		BodyType     string `json:"body_type"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	c := &data.Vehicle{
		LicensePlate: input.LicensePlate,
		Make:         input.Make,
		Model:        input.Model,
		Year:         input.Year,
		Vin:          input.Vin,
		Color:        input.Color,
		BodyType:     input.BodyType,
	}

	data.ValidateVehicle(v, c)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Vehicles.Insert(c)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/vehicles/%d", c.ID))

	err = app.writeJSON(w, http.StatusOK, envelope{"vehicle": c}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	vehicle, err := app.models.Vehicles.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrNoRecordFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"vehicle": vehicle}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	vehicle, err := app.models.Vehicles.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrNoRecordFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if ok := app.checkVersion(r, vehicle.Version); !ok {
		app.editConflictResponse(w, r)
		return
	}

	var input struct {
		LicensePlate *string `json:"license_plate"`
		Make         *string `json:"make"`
		Model        *string `json:"model"`
		Year         *int    `json:"year"`
		Vin          *string `json:"vin"`
		Color        *string `json:"color"`
		BodyType     *string `json:"body_type"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.LicensePlate != nil {
		vehicle.LicensePlate = *input.LicensePlate
	}
	if input.Make != nil {
		vehicle.Make = *input.Make
	}
	if input.Model != nil {
		vehicle.Model = *input.Model
	}
	if input.Year != nil {
		vehicle.Year = *input.Year
	}
	if input.Vin != nil {
		vehicle.Vin = *input.Vin
	}
	if input.Color != nil {
		vehicle.Color = *input.Color
	}
	if input.BodyType != nil {
		vehicle.BodyType = *input.BodyType
	}

	v := validator.New()

	if data.ValidateVehicle(v, vehicle); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Vehicles.Update(vehicle)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"vehicle": vehicle}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Vehicles.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "vehicle successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	v := validator.New()

	var input struct {
		data.Filters
	}

	input.Page = app.readInt(&qs, "page", 1, v)
	input.PageSize = app.readInt(&qs, "page_size", 20, v)
	input.Sort = app.readString(&qs, "sort", "id")

	input.SortSafelist = []string{"year", "color", "body_type", "created_at", "-year", "-color", "-body_type", "-created_at"}

	if data.ValidateFilter(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	vehicles, metadata, err := app.models.Vehicles.GetAll(input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"vehicles": vehicles, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
