package api

import (
	"time"

	"github.com/SEC-Jobstreet/backend-candidate-service/api/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (s *Server) setupRouter() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "content-type", "accept", "authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authRoutes := router.Group("/api/v1").Use(middleware.IsAuthorizedJWT(s.config))

	authRoutes.POST("/apply_job", s.example)

	// Oauth
	apiOauthGoogle := router.Group("/oauth")
	{
		apiOauthGoogle.GET("/:provider/callback", s.authHandler.HandleCallback)
		apiOauthGoogle.GET("/:provider", s.authHandler.HandleAuth)
		apiOauthGoogle.POST("/:provider/refresh_token", s.authHandler.HandleRefresh)
	}

	// Test
	apiHome := router.Group("/test")
	{
		apiHome.GET("/apply_job", middleware.IsAuthorizedJWT(s.config), s.example)
	}

	// Candidate
	apiProfile := router.Group("/profiles")
	{
		apiProfile.GET("", s.apiMiddleware.AuthMiddleware(s.config), s.candidateProfileHandler.GetProfile)
		apiProfile.PUT("", s.apiMiddleware.AuthMiddleware(s.config), s.candidateProfileHandler.UpdateProfile)
	}

	s.router = router
}
