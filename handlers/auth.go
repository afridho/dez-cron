package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/afridhozega/dez-cron/db"
	"github.com/afridhozega/dez-cron/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RegisterWebRoutes(r *gin.Engine) {
	r.GET("/", ShowLogin)
	r.POST("/login", ProcessLogin)
	r.GET("/dashboard", requireCookie(), ShowDashboard)
	r.POST("/generate-token", requireCookie(), GenerateToken)
	r.POST("/revoke-token", requireCookie(), RevokeToken)
	r.GET("/logout", Logout)

	r.GET("/docs", ShowDocs)
	r.GET("/api-docs.yaml", ServeSpec)
}

func requireCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("auth")
		if err != nil || cookie != "logged_in" {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetAPITokens() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var conf models.SysConfig
	err := db.DB.Collection("configs").FindOne(ctx, bson.M{"_id": "global"}).Decode(&conf)
	if err != nil || conf.Tokens == nil {
		return make([]string, 0)
	}
	return conf.Tokens
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokens := GetAPITokens()
		if len(tokens) == 0 {
			// Open API if no tokens generated yet.
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization format. Expected: Bearer <token>"})
			c.Abort()
			return
		}

		valid := false
		for _, t := range tokens {
			if parts[1] == t {
				valid = true
				break
			}
		}

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func generateRandomToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "fallback_token_123456"
	}
	return hex.EncodeToString(bytes)
}

func ShowLogin(c *gin.Context) {
	cookie, err := c.Cookie("auth")
	if err == nil && cookie == "logged_in" {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Login - Dez Cron Admin</title>
	<!-- Favicon -->
	<link rel="icon" type="image/png" href="/assets/favicon.png">
	<style>body{font-family:sans-serif;background:#f4f4f5;display:flex;justify-content:center;align-items:center;height:100vh;}
	.box{background:white;padding:30px;border-radius:8px;box-shadow:0 4px 6px rgba(0,0,0,0.1);text-align:center;}
	input{display:block;margin:10px auto;padding:10px;width:100%;box-sizing:border-box;}
	button{background:#000;color:#fff;padding:10px 20px;border:none;cursor:pointer;width:100%;}
	</style></head>
	<body>
		<div class="box">
			<h2>Dez Cron Login</h2>
			<form action="/login" method="POST">
				<input type="text" name="username" placeholder="Username" required />
				<input type="password" name="password" placeholder="Password" required />
				<button type="submit">Login</button>
			</form>
		</div>
	</body>
	</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func ProcessLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	adminAuth := os.Getenv("ADMIN_AUTH")
	if adminAuth == "" {
		adminAuth = "admin:admin"
	}

	parts := strings.Split(adminAuth, ":")
	expectedUser := "admin"
	expectedPass := "admin"
	if len(parts) == 2 {
		expectedUser = parts[0]
		expectedPass = parts[1]
	}

	if username == expectedUser && password == expectedPass {
		c.SetCookie("auth", "logged_in", 3600*24, "/", "", false, true)
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}
	c.Data(http.StatusUnauthorized, "text/html; charset=utf-8", []byte("<h1>Invalid Credentials</h1><a href='/'>Back</a>"))
}

func Logout(c *gin.Context) {
	c.SetCookie("auth", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func ShowDashboard(c *gin.Context) {
	tokens := GetAPITokens()
	var tokenListHTML string

	if len(tokens) == 0 {
		tokenListHTML = "<p style='color:#ef4444;'>No tokens generated yet. API is completely open! Generate one below.</p>"
	} else {
		for _, t := range tokens {
			tokenListHTML += fmt.Sprintf(`
				<div class="token-box">
					<span>%s</span>
					<form action="/revoke-token" method="POST" style="display:inline; float:right;">
						<input type="hidden" name="token" value="%s" />
						<button type="submit" style="background:#ef4444; padding:5px 10px; margin-top:-5px;">Revoke</button>
					</form>
				</div>
			`, t, t)
		}
	}

	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head><title>Dashboard - Dez Cron</title>
	<!-- Favicon -->
	<link rel="icon" type="image/png" href="/assets/favicon.png">
	<style>body{font-family:sans-serif;background:#f4f4f5;padding:50px;}
	.box{background:white;padding:30px;border-radius:8px;box-shadow:0 4px 6px rgba(0,0,0,0.1);max-width:600px;margin:auto;}
	.token-box{background:#f1f5f9;padding:15px;font-family:monospace;word-break:break-all;border:1px dashed #cbd5e1;margin:10px 0;}
	button{background:#000;color:#fff;padding:10px 20px;border:none;cursor:pointer;}
	a{color:#ef4444;text-decoration:none;float:right;}
	</style></head>
	<body>
		<div class="box">
			<a href="/logout">Logout</a>
			<h2>🔐 API Access Tokens</h2>
			<p>Use any of these tokens via <strong>Authorization: Bearer &lt;token&gt;</strong> to access the API.</p>
			%s
			<form action="/generate-token" method="POST" style="margin-top:20px;">
				<button type="submit">Generate New Token</button>
			</form>
		</div>
	</body>
	</html>`, tokenListHTML)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func GenerateToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newToken := generateRandomToken()
	
	opts := options.Update().SetUpsert(true)
	update := bson.M{"$push": bson.M{"tokens": newToken}}
	_, err := db.DB.Collection("configs").UpdateOne(ctx, bson.M{"_id": "global"}, update, opts)
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte("<h1>Error saving token to DB</h1>"))
		return
	}

	c.Redirect(http.StatusFound, "/dashboard")
}

func RevokeToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenToRevoke := c.PostForm("token")
	if tokenToRevoke == "" {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	update := bson.M{"$pull": bson.M{"tokens": tokenToRevoke}}
	_, err := db.DB.Collection("configs").UpdateOne(ctx, bson.M{"_id": "global"}, update)
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte("<h1>Error revoking token</h1>"))
		return
	}

	c.Redirect(http.StatusFound, "/dashboard")
}
