package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// 默认管理员用户名和密码（生产环境应从环境变量读取）
	DefaultUsername = "admin"
	DefaultPassword = "admin123" // 默认密码，建议修改

	// Session cookie名称
	SessionCookieName = "doc_scanner_session"
	SessionMaxAge     = 3600 * 24 // 24小时
)

// 简单的内存session存储
var activeSessions = make(map[string]time.Time)

// getAdminCredentials 从环境变量获取管理员凭据
func getAdminCredentials() (string, string) {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = DefaultUsername
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = DefaultPassword
	}

	return username, password
}

// hashPassword 对密码进行SHA256哈希
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// generateSessionToken 生成session token
func generateSessionToken(username string) string {
	timestamp := time.Now().UnixNano()
	data := username + string(rune(timestamp))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// cleanExpiredSessions 清理过期的session
func cleanExpiredSessions() {
	now := time.Now()
	for token, expiry := range activeSessions {
		if now.After(expiry) {
			delete(activeSessions, token)
		}
	}
}

// RequireAuth 认证中间件
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 清理过期session
		cleanExpiredSessions()

		// 检查session cookie
		sessionToken, err := c.Cookie(SessionCookieName)
		if err == nil {
			// 验证session是否存在且未过期
			if expiry, exists := activeSessions[sessionToken]; exists {
				if time.Now().Before(expiry) {
					// Session有效，继续处理
					c.Next()
					return
				}
				// Session过期，删除
				delete(activeSessions, sessionToken)
			}
		}

		// 未登录，重定向到登录页
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
	}
}

// RequireAuthAPI API认证中间件（返回JSON而不是重定向）
func RequireAuthAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 清理过期session
		cleanExpiredSessions()

		// 检查session cookie
		sessionToken, err := c.Cookie(SessionCookieName)
		if err == nil {
			// 验证session是否存在且未过期
			if expiry, exists := activeSessions[sessionToken]; exists {
				if time.Now().Before(expiry) {
					// Session有效，继续处理
					c.Next()
					return
				}
				// Session过期，删除
				delete(activeSessions, sessionToken)
			}
		}

		// 未登录，返回401错误
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
			"message": "请先登录",
		})
		c.Abort()
	}
}

// Login 登录处理
func Login(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"message": "用户名和密码不能为空",
		})
		return
	}

	// 获取管理员凭据
	adminUsername, adminPassword := getAdminCredentials()

	// 验证用户名和密码
	if loginReq.Username != adminUsername || loginReq.Password != adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
			"message": "用户名或密码错误",
		})
		return
	}

	// 生成session token
	sessionToken := generateSessionToken(loginReq.Username)
	expiry := time.Now().Add(SessionMaxAge * time.Second)

	// 存储session
	activeSessions[sessionToken] = expiry

	// 设置cookie
	c.SetCookie(
		SessionCookieName,
		sessionToken,
		SessionMaxAge,
		"/",
		"",
		false, // 生产环境应设置为true（HTTPS）
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登录成功",
		"username": loginReq.Username,
	})
}

// Logout 登出处理
func Logout(c *gin.Context) {
	// 获取session cookie
	sessionToken, err := c.Cookie(SessionCookieName)
	if err == nil {
		// 删除session
		delete(activeSessions, sessionToken)
	}

	// 清除cookie
	c.SetCookie(
		SessionCookieName,
		"",
		-1,
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登出成功",
	})
}

// CheckAuth 检查认证状态
func CheckAuth(c *gin.Context) {
	// 清理过期session
	cleanExpiredSessions()

	// 检查session cookie
	sessionToken, err := c.Cookie(SessionCookieName)
	if err == nil {
		// 验证session是否存在且未过期
		if expiry, exists := activeSessions[sessionToken]; exists {
			if time.Now().Before(expiry) {
				c.JSON(http.StatusOK, gin.H{
					"authenticated": true,
					"username": "admin",
				})
				return
			}
			// Session过期，删除
			delete(activeSessions, sessionToken)
		}
	}

	c.JSON(http.StatusUnauthorized, gin.H{
		"authenticated": false,
		"message": "未登录",
	})
}
