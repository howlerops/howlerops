package server

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/internal/services"
)

// registerOrganizationRoutes registers HTTP routes for organization operations
func registerOrganizationRoutes(router *mux.Router, services *services.Services, authMiddleware *middleware.AuthMiddleware, logger *logrus.Logger) {
	// Create organization handler
	orgHandler := organization.NewHandler(services.Organization, logger)

	// Create organization router with auth middleware
	// All organization routes require authentication
	httpAuthMiddleware := middleware.HTTPAuthMiddleware(authMiddleware, logger)

	// Create a subrouter for authenticated organization routes
	orgRouter := router.NewRoute().Subrouter()
	orgRouter.Use(httpAuthMiddleware)

	// Register all organization routes
	orgHandler.RegisterRoutes(orgRouter)

	logger.Info("Organization HTTP routes registered successfully")
}
