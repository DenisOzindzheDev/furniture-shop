package router

import (
	"database/sql"
	"net/http"

	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
	"github.com/DenisOzindzheDev/furniture-shop/internal/transport/http/handler"

	// "github.com/DenisOzindzheDev/furniture-shop/internal/transport/http/handler"

	_ "github.com/DenisOzindzheDev/furniture-shop/docs"
	"github.com/go-redis/redis/v8"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func New(cfg *config.Config, db *sql.DB, redisClient *redis.Client, jwtManager *auth.JWTManager,
	userService *service.UserService, productService *service.ProductService, pdfService *service.PDFService) http.Handler {

	mux := http.NewServeMux()

	userHandler := handler.NewUserHandler(userService)
	productHandler := handler.NewProductHandler(productService)
	productAdminHandler := handler.NewProductAdminHandler(productService)
	productPDFHandler := handler.NewProductPDFHandler(productService, pdfService)
	healthHandler := handler.NewHealthHandler(db, redisClient, nil)

	// Swagger
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Public routes
	mux.HandleFunc("GET /api/health", healthHandler.HealthCheck)
	mux.HandleFunc("POST /api/register", userHandler.Register)
	mux.HandleFunc("POST /api/login", userHandler.Login)
	mux.HandleFunc("GET /api/products", productHandler.ListProducts)
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProduct)
	mux.HandleFunc("GET /api/products/{id}/download", productPDFHandler.DownloadProductPDF)
	mux.HandleFunc("GET /api/products/{id}/preview", productPDFHandler.PreviewProductPDF)

	// Auth middleware
	authMiddleware := auth.AuthMiddleware(jwtManager)
	mux.Handle("GET /api/profile", authMiddleware(http.HandlerFunc(userHandler.Profile)))

	// Admin middleware
	adminMiddleware := auth.AuthMiddleware(jwtManager)
	mux.Handle("POST /api/admin/products", adminMiddleware(http.HandlerFunc(productAdminHandler.CreateProduct)))
	mux.Handle("PUT /api/admin/products/{id}", adminMiddleware(http.HandlerFunc(productAdminHandler.UpdateProduct)))
	mux.Handle("DELETE /api/admin/products/{id}", adminMiddleware(http.HandlerFunc(productAdminHandler.DeleteProduct)))
	mux.Handle("GET /api/admin/products", adminMiddleware(http.HandlerFunc(productAdminHandler.ListProducts)))

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            cfg.CorsDebug,
	})

	return c.Handler(mux)
}
