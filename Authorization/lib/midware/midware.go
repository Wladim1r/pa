// Package midware
package midware

import (
	"context"
	"errors"
	"net"
	"net/http"

	repo "github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/Wladim1r/auth/periferia/reddis"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func CheckCookie(ctx context.Context, rdb *reddis.RDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"no cookie": err.Error(),
			})
			c.Abort()
			return
		}

		name, err := rdb.Receive(ctx, token)
		if err != nil {
			if err == redis.Nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "cookie not found",
				})
			}
			if _, ok := err.(net.Error); ok {
				// Network error, use fallback
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "error on the server side: " + err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "invalid cookie",
				})
			}

			c.Abort()
			return
		}

		c.Set("token", token)
		c.Set("username", name)
		c.Next()
	}
}

func CheckUserExists(db repo.UsersDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		password, err := db.SelectPwdByName(req.Name)
		if err != nil {
			switch {
			case errors.Is(err, errs.ErrRecordingWNF):
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "user does not registered ❌",
				})
				c.Abort()
				return
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "db error: " + err.Error(),
				})
				c.Abort()
				return
			}
		}

		if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": "passwords not equal 🚫🟰",
			})
			c.Abort()
		}

		c.Set("username", req.Name)
		c.Next()
	}
}
