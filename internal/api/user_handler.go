package api

import (
	"encoding/json"
	"errors"
	"go-backend/internal/model"
	"go-backend/internal/service"
	"net/http"
)

func SignUpHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Name     string `json:"name" validate:"required"`
			Password string `json:"password" validate:"required,min=8,max=72"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if err := validate.Struct(req); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		user, err := svc.SignUp(r.Context(), model.User{
			Email:    req.Email,
			Name:     req.Name,
			Password: req.Password,
		})

		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusCreated, user)
	}
}

func LogInHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required,min=8,max=72"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if err := validate.Struct(req); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		tokenString, err := svc.LogIn(r.Context(), req.Email, req.Password)
		if mapServiceError(w, err) {
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   3600,
		})
		respondJSON(w, http.StatusOK, map[string]string{"message": "logged in"})
	}
}

func GetUserHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		currentUserID := r.Context().Value(UserIDKey).(int64)
		isAdmin, _ := r.Context().Value(IsAdminKey).(bool)
		if currentUserID != id && !isAdmin {
			respondError(w, http.StatusForbidden, "access denied")
			return
		}

		user, err := svc.GetUser(r.Context(), id)
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, user)
	}
}

func GetAllUsersHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := svc.GetAllUsers(r.Context())
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, users)
	}
}
