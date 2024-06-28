package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFoundResponse(w, r)
	})

	router.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.methodNotAllowedResponse(w, r)
	})

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/vehicles", app.listVehiclesHandler)
	router.HandlerFunc(http.MethodGet, "/v1/vehicles/:id", app.showVehiclesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/vehicles", app.createVehiclesHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/vehicles/:id", app.updateVehiclesHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/vehicles/:id", app.deleteVehiclesHandler)

	// User and Auth
	router.HandlerFunc(http.MethodPut, "/v1/users/resetpassword", app.resetPasswordHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/updatepassword", app.updatePasswordHandler)
	router.HandlerFunc(http.MethodPost, "/v1/users/register", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	router.HandlerFunc(http.MethodPut, "/v1/tokens/refresh", app.requireAuthentication(app.refreshTokenHandler))

	standard := alice.New(app.metrics, app.recoverPanic, app.enableCORS, app.rateLimiter, app.authenticate)

	return standard.Then(router)
}
