package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/data"
	"github.com/PriyanshuSharma23/follow-ups-server/internals/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var inp struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &inp)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var user = &data.User{
		Name:      inp.Name,
		Email:     inp.Email,
		Activated: false,
	}

	err = user.Password.Set(inp.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var v = validator.New()
	data.ValidateUser(v, user)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.models.Tokens.New(user.ID, time.Hour*24*3, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"userId":          user.ID,
			"activationToken": token.Plaintext,
		}

		err := app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, map[string]string{
				"email": user.Email,
			})
		}
	})

	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidatePlaintextToken(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetForToken(input.TokenPlaintext, data.ScopeActivation)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			v.AddError("token", "invalid or expired token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user.Activated = true

	err = app.models.Users.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.Tokens.DeleteAllForUser(user.ID, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var v = validator.New()
	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrNoRecordFound) {
			err = app.writeJSON(w, http.StatusOK, envelope{"message": "check your registered email address"}, nil)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
		} else {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	token, err := app.models.Tokens.New(user.ID, time.Hour*3, data.ScopePasswordReset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"resetPasswordToken": token.Plaintext,
		}
		err = app.mailer.Send(input.Email, "reset_password.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, map[string]string{
				"email": user.Email,
			})
		}
	})

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "check your registered email address"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password string `json:"password"`
		Token    string `json:"token"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.models.Users.GetForToken(input.Token, data.ScopePasswordReset)
	if err != nil {
		if errors.Is(err, data.ErrNoRecordFound) {
			app.invalidTokenResponse(w, r, "password-reset")
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user.Password.Set(input.Password)

	var v = validator.New()
	data.ValidateUser(v, user)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.UpdateUser(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.models.Tokens.ClearAllForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
