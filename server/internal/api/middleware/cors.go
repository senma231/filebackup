package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		_ = c.Request.Header.Get("Origin")  // origin 变量已声明但未使用

		// 允许所有来源访问
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, X-Token, X-User-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")

		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
