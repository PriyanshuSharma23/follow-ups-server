package main

import "net/http"

func (app *application) errorResponse(w http.ResponseWriter, _ *http.Request, status int, data any) {
	env := envelope{
		"error": data,
	}

	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logger.PrintError(err, nil)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.PrintError(err, nil)

	message := `something went wrong while processing the request`
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}
