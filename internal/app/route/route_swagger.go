package route

import (
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
	"github.com/frontleaves-mc/frontleaves-plugin/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func swaggerRegister(r gin.IRouter) {
	docs.SwaggerInfofrontleaves_plugin.BasePath = "/api/v1"
	docs.SwaggerInfofrontleaves_plugin.Title = "Bamboo Base Go Template"
	docs.SwaggerInfofrontleaves_plugin.Description = "bamboo-base-go-template API 文档"
	docs.SwaggerInfofrontleaves_plugin.Version = "v1.0.0"
	docs.SwaggerInfofrontleaves_plugin.Host = xEnv.GetEnvString(xEnv.Host, "localhost") + ":" + xEnv.GetEnvString(xEnv.Port, "8080")
	docs.SwaggerInfofrontleaves_plugin.Schemes = []string{"http", "https"}

	swaggerGroup := r.Group("/swagger")
	swaggerGroup.GET("/plugin/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("frontleaves_plugin")))
}
