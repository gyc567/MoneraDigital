// internal/docs/swagger.go
package docs

import (
	"github.com/swaggo/swag"
)

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

func init() {
	swag.Register(SwaggerInfo.InstanceName(), &swag.Spec{
		Version:          SwaggerInfo.Version,
		Host:             SwaggerInfo.Host,
		BasePath:         SwaggerInfo.BasePath,
		Schemes:          SwaggerInfo.Schemes,
		Title:            SwaggerInfo.Title,
		Description:      SwaggerInfo.Description,
		InfoInstanceName: "swagger",
		SwaggerTemplate:  swaggerTemplate,
	})
}

const swaggerTemplate = `{
  "swagger": "2.0",
  "info": {
    "title": "MoneraDigital API",
    "description": "Institutional-grade digital asset platform API",
    "version": "1.0"
  },
  "host": "api.monera-digital.com",
  "basePath": "/api",
  "schemes": ["https"],
  "consumes": ["application/json"],
  "produces": ["application/json"],
  "paths": {
    "/auth/register": {
      "post": {
        "summary": "Register a new user",
        "description": "Create a new user account with email and password",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/RegisterRequest"
            }
          }
        ],
        "responses": {
          "201": {
            "description": "User created successfully",
            "schema": {
              "$ref": "#/definitions/UserInfo"
            }
          },
          "400": {
            "description": "Invalid request",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "409": {
            "description": "Email already registered",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    },
    "/auth/login": {
      "post": {
        "summary": "User login",
        "description": "Authenticate user and receive access and refresh tokens",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/LoginRequest"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Login successful",
            "schema": {
              "$ref": "#/definitions/LoginResponse"
            }
          },
          "400": {
            "description": "Invalid request",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "401": {
            "description": "Invalid credentials",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    },
    "/auth/refresh": {
      "post": {
        "summary": "Refresh access token",
        "description": "Use refresh token to obtain a new access token",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/RefreshTokenRequest"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Token refreshed successfully",
            "schema": {
              "$ref": "#/definitions/RefreshTokenResponse"
            }
          },
          "401": {
            "description": "Invalid or revoked refresh token",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    },
    "/auth/logout": {
      "post": {
        "summary": "User logout",
        "description": "Revoke the current access token",
        "security": [
          {
            "Bearer": []
          }
        ],
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/LogoutRequest"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Logout successful"
          },
          "401": {
            "description": "Unauthorized",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    },
    "/lending/apply": {
      "post": {
        "summary": "Apply for lending",
        "description": "Submit a lending application",
        "security": [
          {
            "Bearer": []
          }
        ],
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/ApplyLendingRequest"
            }
          }
        ],
        "responses": {
          "201": {
            "description": "Lending application created",
            "schema": {
              "$ref": "#/definitions/LendingPositionResponse"
            }
          },
          "400": {
            "description": "Invalid request",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "401": {
            "description": "Unauthorized",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    },
    "/lending/positions": {
      "get": {
        "summary": "Get user lending positions",
        "description": "Retrieve all lending positions for the authenticated user",
        "security": [
          {
            "Bearer": []
          }
        ],
        "responses": {
          "200": {
            "description": "List of lending positions",
            "schema": {
              "$ref": "#/definitions/LendingPositionsListResponse"
            }
          },
          "401": {
            "description": "Unauthorized",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "RegisterRequest": {
      "type": "object",
      "required": ["email", "password"],
      "properties": {
        "email": {
          "type": "string",
          "format": "email"
        },
        "password": {
          "type": "string",
          "minLength": 8
        }
      }
    },
    "LoginRequest": {
      "type": "object",
      "required": ["email", "password"],
      "properties": {
        "email": {
          "type": "string",
          "format": "email"
        },
        "password": {
          "type": "string"
        }
      }
    },
    "UserInfo": {
      "type": "object",
      "properties": {
        "id": {
          "type": "integer"
        },
        "email": {
          "type": "string"
        }
      }
    },
    "LoginResponse": {
      "type": "object",
      "properties": {
        "access_token": {
          "type": "string"
        },
        "refresh_token": {
          "type": "string"
        },
        "token_type": {
          "type": "string"
        },
        "expires_in": {
          "type": "integer"
        },
        "expires_at": {
          "type": "string",
          "format": "date-time"
        },
        "user": {
          "$ref": "#/definitions/UserInfo"
        }
      }
    },
    "RefreshTokenRequest": {
      "type": "object",
      "required": ["refresh_token"],
      "properties": {
        "refresh_token": {
          "type": "string"
        }
      }
    },
    "RefreshTokenResponse": {
      "type": "object",
      "properties": {
        "access_token": {
          "type": "string"
        },
        "token_type": {
          "type": "string"
        },
        "expires_in": {
          "type": "integer"
        },
        "expires_at": {
          "type": "string",
          "format": "date-time"
        }
      }
    },
    "LogoutRequest": {
      "type": "object",
      "required": ["token"],
      "properties": {
        "token": {
          "type": "string"
        }
      }
    },
    "ApplyLendingRequest": {
      "type": "object",
      "required": ["asset", "amount", "duration_days"],
      "properties": {
        "asset": {
          "type": "string",
          "enum": ["BTC", "ETH", "USDC", "USDT"]
        },
        "amount": {
          "type": "number",
          "format": "double"
        },
        "duration_days": {
          "type": "integer",
          "minimum": 1,
          "maximum": 365
        }
      }
    },
    "LendingPositionResponse": {
      "type": "object",
      "properties": {
        "id": {
          "type": "integer"
        },
        "user_id": {
          "type": "integer"
        },
        "asset": {
          "type": "string"
        },
        "amount": {
          "type": "number"
        },
        "duration_days": {
          "type": "integer"
        },
        "apy": {
          "type": "number"
        },
        "status": {
          "type": "string"
        },
        "accrued_yield": {
          "type": "number"
        },
        "start_date": {
          "type": "string",
          "format": "date-time"
        },
        "end_date": {
          "type": "string",
          "format": "date-time"
        },
        "created_at": {
          "type": "string",
          "format": "date-time"
        }
      }
    },
    "LendingPositionsListResponse": {
      "type": "object",
      "properties": {
        "positions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/LendingPositionResponse"
          }
        },
        "total": {
          "type": "integer"
        },
        "count": {
          "type": "integer"
        }
      }
    },
    "ErrorResponse": {
      "type": "object",
      "properties": {
        "code": {
          "type": "string"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "string"
        }
      }
    }
  },
  "securityDefinitions": {
    "Bearer": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header",
      "description": "JWT Authorization header using the Bearer scheme"
    }
  }
}
`
