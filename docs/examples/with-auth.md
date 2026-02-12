# Adding JWT Authentication

This example shows how to protect GORM Studio with JWT-based authentication.

## Overview

GORM Studio does not include built-in authentication. Instead, you can use Gin's middleware system to add your own authentication layer. This example demonstrates three approaches:

1. Basic authentication
2. JWT authentication
3. Session-based authentication

## Approach 1: Basic Auth

```go
package main

import (
    "github.com/MUKE-coder/gorm-studio/studio"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
    router := gin.Default()

    // Protect the studio prefix with basic auth
    studioGroup := router.Group("/studio", gin.BasicAuth(gin.Accounts{
        "admin":   "secure-password-here",
        "viewer":  "another-password",
    }))

    // Since studio.Mount() expects *gin.Engine, we register routes manually
    // For basic auth, the simplest approach is to wrap the whole engine:
    authedRouter := gin.Default()
    authedRouter.Use(gin.BasicAuth(gin.Accounts{
        "admin": "secure-password-here",
    }))

    studio.Mount(authedRouter, db, []interface{}{})
    authedRouter.Run(":8080")
}
```

## Approach 2: JWT Authentication

### Install Dependencies

```bash
go get github.com/golang-jwt/jwt/v5
```

### JWT Middleware

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/MUKE-coder/gorm-studio/studio"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

// JWTAuthMiddleware validates JWT tokens
func JWTAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            // For browser access, check for token in cookie
            tokenCookie, err := c.Cookie("studio_token")
            if err != nil {
                c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                    "error": "Authorization required",
                })
                return
            }
            authHeader = "Bearer " + tokenCookie
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return jwtSecret, nil
        })

        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid or expired token",
            })
            return
        }

        // Extract claims
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Set("user_id", claims["sub"])
            c.Set("role", claims["role"])
        }

        c.Next()
    }
}

// AdminOnly requires the "admin" role
func AdminOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        role, exists := c.Get("role")
        if !exists || role != "admin" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "Admin access required",
            })
            return
        }
        c.Next()
    }
}

// GenerateToken creates a JWT token (call this from your login endpoint)
func GenerateToken(userID string, role string) (string, error) {
    claims := jwt.MapClaims{
        "sub":  userID,
        "role": role,
        "exp":  time.Now().Add(24 * time.Hour).Unix(),
        "iat":  time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"size:100"`
    Email string `gorm:"size:200"`
}

func main() {
    db, err := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    db.AutoMigrate(&User{})

    router := gin.Default()

    // Public routes
    router.POST("/login", func(c *gin.Context) {
        // Your login logic here...
        token, _ := GenerateToken("user-1", "admin")
        c.JSON(200, gin.H{"token": token})
    })

    // Protected studio with JWT + admin role
    protectedRouter := gin.Default()
    protectedRouter.Use(JWTAuthMiddleware(), AdminOnly())

    studio.Mount(protectedRouter, db, []interface{}{&User{}}, studio.Config{
        Prefix: "/studio",
    })

    protectedRouter.Run(":8080")
}
```

### Frontend Token Handling

Since the GORM Studio frontend makes API calls from the browser, you'll need the token available. Options:

1. **Cookie-based** — Set an `HttpOnly` cookie after login; the middleware checks cookies as a fallback
2. **Query parameter** — Pass token as `?token=...` on initial page load (less secure)
3. **Proxy** — Use a reverse proxy that handles auth and forwards requests

## Approach 3: Session-Based Auth

```go
import "github.com/gin-contrib/sessions"
import "github.com/gin-contrib/sessions/cookie"

func SessionAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := sessions.Default(c)
        userID := session.Get("user_id")
        if userID == nil {
            c.Redirect(http.StatusFound, "/login")
            c.Abort()
            return
        }
        c.Next()
    }
}

func main() {
    router := gin.Default()

    store := cookie.NewStore([]byte("secret"))
    router.Use(sessions.Sessions("session", store))

    // Login page sets session
    router.POST("/login", func(c *gin.Context) {
        // Validate credentials...
        session := sessions.Default(c)
        session.Set("user_id", 1)
        session.Save()
        c.Redirect(http.StatusFound, "/studio")
    })

    // Protected studio
    protectedRouter := gin.Default()
    protectedRouter.Use(sessions.Sessions("session", store), SessionAuth())

    studio.Mount(protectedRouter, db, models)
    protectedRouter.Run(":8080")
}
```

## Security Best Practices

1. **Never hardcode secrets** — Use environment variables for JWT secrets and passwords
2. **Use HTTPS** — Always use TLS in non-localhost environments
3. **Set token expiration** — Short-lived tokens (1-24 hours) reduce risk
4. **Log access** — Add logging middleware to track who accesses the studio
5. **Combine with read-only** — Use `ReadOnly: true` for non-admin users

```go
// Read-only for regular users, full access for admins
func RoleBasedConfig(c *gin.Context) studio.Config {
    role, _ := c.Get("role")
    if role == "admin" {
        return studio.Config{ReadOnly: false, DisableSQL: false}
    }
    return studio.Config{ReadOnly: true, DisableSQL: true}
}
```
