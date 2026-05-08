package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go-backend/internal/model"
	"go-backend/internal/store"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World! You requested: %s", r.URL.Path)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func CreateUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user model.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if user.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required")
			return
		}

		id, err := userStore.Save(user)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create user")
			return
		}

		user.ID = id
		respondJSON(w, http.StatusCreated, user)
	}
}

func GetUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		idInt, err := strconv.Atoi(id)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		user, err := userStore.Get(idInt)
		if err != nil {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}

		respondJSON(w, http.StatusOK, user)
	}
}

func GetAllUsersHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := userStore.GetAll()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to retrieve users")
			return
		}

		respondJSON(w, http.StatusOK, users)
	}
}

func DeleteUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		idInt, err := strconv.Atoi(id)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		err = userStore.Delete(idInt)
		if err != nil {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
