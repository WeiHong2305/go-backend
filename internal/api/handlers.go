package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go-backend/internal/model"
	"go-backend/internal/store"
)

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
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if user.Name == "" {
			http.Error(w, "missing required fields (name)", http.StatusBadRequest)
			return
		}

		pk := userStore.Save(user)

		fmt.Printf("ID = %d\n", pk)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

func GetUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		idInt, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}

		user, ok := userStore.Get(idInt)
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)

		fmt.Printf("Name: %s\n", user.Name)
		fmt.Printf("Active: %t\n", user.Active)
	}
}

func GetAllUsersHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users := userStore.GetAll()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(users)
	}
}

func DeleteUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		idInt, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}

		if _, ok := userStore.Get(idInt); !ok {
			http.NotFound(w, r)
			return
		}

		userStore.Delete(idInt)
		w.WriteHeader(http.StatusNoContent)
	}
}
