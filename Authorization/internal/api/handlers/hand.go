// Package handlers
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Wladim1r/auth/internal/api/service"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/Wladim1r/auth/lib/hashpwd"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type handler struct {
	ctx context.Context
	s   service.Service
}

func NewHandler(ctx context.Context, service service.Service) *handler {
	return &handler{
		ctx: ctx,
		s:   service,
		// rdb: rdb,
	}
}

func (h *handler) Registration(c *gin.Context) {
	var req models.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.s.CheckUserExists(req.Name)

	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			hashPwd, err := hashpwd.HashPwd([]byte(req.Password), req.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"Could not hash password": err.Error(),
				})
				return
			}

			err = h.s.CreateUser(req.Name, hashPwd)
			if err != nil {
				switch {
				case errors.Is(err, errs.ErrRecordingWNC):
					c.JSON(http.StatusInternalServerError, gin.H{
						"Could not create user rawsAffected=0": err.Error(),
					})
					return

				default:
					c.JSON(http.StatusInternalServerError, gin.H{
						"Could not create user": err.Error(),
					})
					return
				}
			}
			c.JSON(http.StatusCreated, gin.H{
				"message": "user successful created 🎊🤩",
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "db error: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusConflict, gin.H{
		"message": "user already exsited 💩",
	})
}

func createJWT(name string) (string, error) {
	claims := jwt.MapClaims{
		"sub": name,
		"exp": time.Now().Add(40 * time.Second).Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := jwtToken.SignedString(getenv.GetString("SECRET_KEY", "default_secret_key"))
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt: %w", err)
	}

	return signedToken, nil
}

func (h *handler) Login(c *gin.Context) {
	name, ok := getFromCtx(c, "username")
	if !ok {
		return
	}

	jwt, err := createJWT(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":   "Login success!🫦",
		"token": jwt,
	})
}

func (h *handler) Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "molodec! 👍",
	})
}

func (h *handler) Delacc(c *gin.Context) {
	name, ok := getFromCtx(c, "username")
	if !ok {
		return
	}

	err := h.s.DeleteUser(name)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWND):
			c.JSON(http.StatusInternalServerError, gin.H{
				"Could not create user rawsAffected=0": err.Error(),
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "❌🗑️ Could not delete user: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "👍 user has successful deleted from DB",
	})
}

func getFromCtx(c *gin.Context, key string) (string, bool) {
	username, exists := c.Get(key)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": fmt.Sprintf("context var %s does not exist", key),
		})
		return "", false
	}
	name := username.(string)

	return name, true
}
