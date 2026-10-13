package routes

import (
	"ty-notebook-api/internal/api/handlers"
	"ty-notebook-api/internal/api/response"
	"ty-notebook-api/internal/database"
	"ty-notebook-api/internal/repository"
	"ty-notebook-api/internal/service"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine) {
	pingHandler := handlers.NewPingHandler()

	api := r.Group("/api/v1")
	{
		api.GET("/ping", pingHandler.Ping)

		if database.DB == nil {
			registerDatabaseUnavailableProjectRoutes(api)
			return
		}

		projectRepo := repository.NewProjectRepository(database.DB)
		projectService := service.NewProjectService(projectRepo)
		projectHandler := handlers.NewProjectHandler(projectService)
		documentRepo := repository.NewDocumentRepository(database.DB)
		documentService := service.NewDocumentService(documentRepo)
		documentHandler := handlers.NewDocumentHandler(documentService)
		publicLinkRepo := repository.NewPublicLinkRepository(database.DB)
		publicLinkService := service.NewPublicLinkService(publicLinkRepo)
		publicLinkHandler := handlers.NewPublicLinkHandler(publicLinkService)

		projects := api.Group("/projects")
		{
			projects.GET("", projectHandler.List)
			projects.POST("", projectHandler.Create)
			projects.GET("/:id", projectHandler.Get)
			projects.PUT("/:id", projectHandler.Update)
			projects.DELETE("/:id", projectHandler.Delete)
			projects.POST("/:id/restore", projectHandler.Restore)
		}

		documents := api.Group("/documents")
		{
			documents.GET("", documentHandler.List)
			documents.POST("", documentHandler.Save)
			documents.GET("/trash", documentHandler.ListDeleted)
			documents.GET("/project/:projectId/type/:type", documentHandler.GetByProjectAndType)
			documents.GET("/:id/versions", documentHandler.ListVersions)
			documents.GET("/:id/versions/:versionId", documentHandler.GetVersion)
			documents.POST("/:id/versions/:versionId/restore", documentHandler.RestoreDocumentVersion)
			documents.GET("/:id", documentHandler.Get)
			documents.PUT("/:id", documentHandler.Save)
			documents.DELETE("/:id", documentHandler.Delete)
			documents.POST("/:id/restore", documentHandler.Restore)
		}

		documentVersions := api.Group("/document-versions")
		{
			documentVersions.POST("/:versionId/restore", documentHandler.RestoreVersion)
		}

		publicLinks := api.Group("/public-links")
		{
			publicLinks.GET("", publicLinkHandler.List)
			publicLinks.POST("", publicLinkHandler.Create)
			publicLinks.PUT("/:id", publicLinkHandler.Update)
			publicLinks.DELETE("/:id", publicLinkHandler.Delete)
			publicLinks.POST("/:id/restore", publicLinkHandler.Restore)
		}
	}
}

func registerDatabaseUnavailableProjectRoutes(api *gin.RouterGroup) {
	unavailable := func(c *gin.Context) {
		response.Error(c, 500, "database is not initialized")
	}

	projects := api.Group("/projects")
	{
		projects.GET("", unavailable)
		projects.POST("", unavailable)
		projects.GET("/:id", unavailable)
		projects.PUT("/:id", unavailable)
		projects.DELETE("/:id", unavailable)
		projects.POST("/:id/restore", unavailable)
	}

	documents := api.Group("/documents")
	{
		documents.GET("", unavailable)
		documents.POST("", unavailable)
		documents.GET("/trash", unavailable)
		documents.GET("/project/:projectId/type/:type", unavailable)
		documents.GET("/:id/versions", unavailable)
		documents.GET("/:id/versions/:versionId", unavailable)
		documents.POST("/:id/versions/:versionId/restore", unavailable)
		documents.GET("/:id", unavailable)
		documents.PUT("/:id", unavailable)
		documents.DELETE("/:id", unavailable)
		documents.POST("/:id/restore", unavailable)
	}

	documentVersions := api.Group("/document-versions")
	{
		documentVersions.POST("/:versionId/restore", unavailable)
	}

	publicLinks := api.Group("/public-links")
	{
		publicLinks.GET("", unavailable)
		publicLinks.POST("", unavailable)
		publicLinks.PUT("/:id", unavailable)
		publicLinks.DELETE("/:id", unavailable)
		publicLinks.POST("/:id/restore", unavailable)
	}
}
