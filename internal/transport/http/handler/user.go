package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/common/errors"
	"github.com/DenisOzindzheDev/furniture-shop/internal/domain/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type RegisterRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
	Name     string `json:"name" example:"John Doe"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}

type AuthResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  *entity.User `json:"user"`
}

// ErrorUserResponse представляет стандартную структуру ошибки для user-хендлеров
// @Description ErrorUserResponse используется для отображения ошибок API
type ErrorUserResponse struct {
	Code    int    `json:"code" example:"500"`
	Message string `json:"message" example:"Internal server error"`
	Details string `json:"details,omitempty" example:"ошибка при обращении к базе"`
}

// Register godoc
// @Summary Регистрация пользователя
// @Description Создает нового пользователя в системе
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные для регистрации"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} ErrorUserResponse
// @Failure 409 {object} ErrorUserResponse
// @Failure 500 {object} ErrorUserResponse
// @Router /register [post]
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUserError(w, http.StatusBadRequest, "Некорректное тело запроса", err.Error())
		return
	}

	user := &entity.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     "customer",
	}

	token, err := h.userService.Register(r.Context(), user)
	if err != nil {
		switch err {
		case errors.ErrUserExists:
			writeUserError(w, http.StatusConflict, "Пользователь уже существует", err.Error())
		default:
			log.Printf("Register error: %v", err)
			writeUserError(w, http.StatusInternalServerError, "Ошибка при регистрации пользователя", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login godoc
// @Summary Авторизация пользователя
// @Description Выполняет вход пользователя и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Данные для входа"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorUserResponse
// @Failure 401 {object} ErrorUserResponse
// @Failure 500 {object} ErrorUserResponse
// @Router /login [post]
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUserError(w, http.StatusBadRequest, "Некорректное тело запроса", err.Error())
		return
	}

	token, user, err := h.userService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case errors.ErrInvalidCredentials:
			writeUserError(w, http.StatusUnauthorized, "Неверный email или пароль", err.Error())
		default:
			log.Printf("Login error: %v", err)
			writeUserError(w, http.StatusInternalServerError, "Ошибка при входе", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User:  user,
	})
}

// Profile godoc
// @Summary Получение профиля пользователя
// @Description Возвращает информацию о текущем пользователе по JWT токену
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entity.User
// @Failure 401 {object} ErrorUserResponse
// @Failure 500 {object} ErrorUserResponse
// @Router /profile [get]
func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		writeUserError(w, http.StatusUnauthorized, "Неавторизованный доступ", "JWT токен отсутствует или недействителен")
		return
	}

	user, err := h.userService.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("Profile error: %v", err)
		writeUserError(w, http.StatusInternalServerError, "Не удалось получить профиль пользователя", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}
