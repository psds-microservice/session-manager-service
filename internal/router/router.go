package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/psds-microservice/helpy/paths"
	"github.com/psds-microservice/session-manager-service/api"
	"github.com/psds-microservice/session-manager-service/internal/handler"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(sessionHandler *handler.SessionHandler) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET(paths.PathHealth, handler.Health)
	r.GET(paths.PathReady, handler.Ready)
	r.GET(paths.PathSwagger, func(c *gin.Context) { c.Redirect(http.StatusFound, paths.PathSwagger+"/") })
	r.GET(paths.PathSwagger+"/*any", func(c *gin.Context) {
		if strings.TrimPrefix(c.Param("any"), "/") == "openapi.json" {
			c.Data(http.StatusOK, "application/json", api.OpenAPISpec)
			return
		}
		if strings.TrimPrefix(c.Param("any"), "/") == "" {
			c.Request.URL.Path = paths.PathSwagger + "/index.html"
			c.Request.RequestURI = paths.PathSwagger + "/index.html"
		}
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/openapi.json"))(c)
	})

	session := r.Group("/session")
	{
		session.GET("/:id", sessionHandler.GetSession)
		session.GET("/:id/participants", sessionHandler.GetParticipants)
		session.POST("/join", sessionHandler.JoinSession)
		session.POST("/:id/invite", sessionHandler.Invite)
		session.POST("/:id/control", sessionHandler.Control)
	}

	return r
}
