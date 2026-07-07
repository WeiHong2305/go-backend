package testutil

import (
	"context"
	"go-backend/internal/model"
)

type MockMovieRepository struct {
	SaveFunc   func(ctx context.Context, m model.Movie) (int64, error)
	GetFunc    func(ctx context.Context, id int64) (model.Movie, error)
	GetAllFunc func(ctx context.Context) ([]model.Movie, error)
	UpdateFunc func(ctx context.Context, id int64, m model.Movie) (model.Movie, error)
	DeleteFunc func(ctx context.Context, id int64) error
}

func (m *MockMovieRepository) Save(ctx context.Context, movie model.Movie) (int64, error) {
	return m.SaveFunc(ctx, movie)
}

func (m *MockMovieRepository) Get(ctx context.Context, id int64) (model.Movie, error) {
	return m.GetFunc(ctx, id)
}

func (m *MockMovieRepository) GetAll(ctx context.Context) ([]model.Movie, error) {
	return m.GetAllFunc(ctx)
}

func (m *MockMovieRepository) Update(ctx context.Context, id int64, movie model.Movie) (model.Movie, error) {
	return m.UpdateFunc(ctx, id, movie)
}

func (m *MockMovieRepository) Delete(ctx context.Context, id int64) error {
	return m.DeleteFunc(ctx, id)
}

type MockUserRepository struct {
	CreateFunc     func(ctx context.Context, user model.User) (model.User, error)
	GetByEmailFunc func(ctx context.Context, email string) (model.User, error)
	GetByIdFunc    func(ctx context.Context, id int64) (model.User, error)
	GetAllFunc     func(ctx context.Context) ([]model.User, error)
}

func (u *MockUserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	return u.CreateFunc(ctx, user)
}

func (u *MockUserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	return u.GetByEmailFunc(ctx, email)
}

func (u *MockUserRepository) GetById(ctx context.Context, id int64) (model.User, error) {
	return u.GetByIdFunc(ctx, id)
}

func (u *MockUserRepository) GetAll(ctx context.Context) ([]model.User, error) {
	return u.GetAllFunc(ctx)
}

type MockCache struct {
	GetFunc    func(ctx context.Context, key string) (string, error)
	SetFunc    func(ctx context.Context, key string, value string) error
	DeleteFunc func(ctx context.Context, key string) error
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	return m.GetFunc(ctx, key)
}

func (m *MockCache) Set(ctx context.Context, key string, value string) error {
	return m.SetFunc(ctx, key, value)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return m.DeleteFunc(ctx, key)
}

type MockMovieService struct {
	CreateMovieFunc  func(ctx context.Context, movie model.Movie) (model.Movie, error)
	GetMovieFunc     func(ctx context.Context, id int64) (model.Movie, error)
	GetAllMoviesFunc func(ctx context.Context) ([]model.Movie, error)
	UpdateMovieFunc  func(ctx context.Context, id int64, patch model.MoviePatch) (model.Movie, error)
	DeleteMovieFunc  func(ctx context.Context, id int64) error
}

func (m *MockMovieService) CreateMovie(ctx context.Context, movie model.Movie) (model.Movie, error) {
	return m.CreateMovieFunc(ctx, movie)
}

func (m *MockMovieService) GetMovie(ctx context.Context, id int64) (model.Movie, error) {
	return m.GetMovieFunc(ctx, id)
}
func (m *MockMovieService) GetAllMovies(ctx context.Context) ([]model.Movie, error) {
	return m.GetAllMoviesFunc(ctx)
}
func (m *MockMovieService) UpdateMovie(ctx context.Context, id int64, patch model.MoviePatch) (model.Movie, error) {
	return m.UpdateMovieFunc(ctx, id, patch)
}
func (m *MockMovieService) DeleteMovie(ctx context.Context, id int64) error {
	return m.DeleteMovieFunc(ctx, id)
}

type MockUserService struct {
	SignUpFunc      func(ctx context.Context, user model.User) (model.User, error)
	LogInFunc       func(ctx context.Context, email, password string) (string, error)
	GetUserFunc     func(ctx context.Context, id int64) (model.User, error)
	GetAllUsersFunc func(ctx context.Context) ([]model.User, error)
}

func (m *MockUserService) SignUp(ctx context.Context, user model.User) (model.User, error) {
	return m.SignUpFunc(ctx, user)
}

func (m *MockUserService) LogIn(ctx context.Context, email, password string) (string, error) {
	return m.LogInFunc(ctx, email, password)
}
func (m *MockUserService) GetUser(ctx context.Context, id int64) (model.User, error) {
	return m.GetUserFunc(ctx, id)
}
func (m *MockUserService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	return m.GetAllUsersFunc(ctx)
}

type MockJobService struct {
	AddJobFunc func(jobType string, payload model.JobPayload) (model.JobRespond, error)
}

func (m *MockJobService) AddJob(jobType string, payload model.JobPayload) (model.JobRespond, error) {
	return m.AddJobFunc(jobType, payload)
}
