// internal/docs/swagger.go
package docs

import "github.com/swaggo/swag"

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}{
	Version:     "1.0",
	Host:        "api.monera-digital.com",
	BasePath:    "/api",
	Schemes:     []string{"https"},
	Title:       "MoneraDigital API",
	Description: "Institutional-grade digital asset platform API",
}

// swaggerInstanceName is the registered swag instance name
const swaggerInstanceName = "swagger"

// NewSwagger initializes the swagger documentation
func NewSwagger() {
	swag.Register(swaggerInstanceName, &swag.Spec{
		Version:          SwaggerInfo.Version,
		Host:             SwaggerInfo.Host,
		BasePath:         SwaggerInfo.BasePath,
		Schemes:          SwaggerInfo.Schemes,
		Title:            SwaggerInfo.Title,
		Description:      SwaggerInfo.Description,
		InfoInstanceName: swaggerInstanceName,
		SwaggerTemplate:  swaggerTemplate,
	})
}

// GetSwaggerInstanceName returns the swag instance name
func GetSwaggerInstanceName() string {
	return swaggerInstanceName
}
