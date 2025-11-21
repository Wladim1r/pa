// Package handlers
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	repo "github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/Wladim1r/auth/lib/hashpwd"
	"github.com/Wladim1r/auth/periferia/reddis"
	"github.com/gin-gonic/gin"
)

type handler struct {
	ctx  context.Context
	repo repo.UsersDB
	rdb  *reddis.RDB
}

func NewHandler(ctx context.Context, repo repo.UsersDB, rdb *reddis.RDB) *handler {
	return &handler{
		ctx:  ctx,
		repo: repo,
		rdb:  rdb,
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

	err := h.repo.CheckUserExists(req.Name)

	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			hashPwd, err := hashpwd.HashPwd(h.repo, []byte(req.Password), req.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"Could not hash password": err.Error(),
				})
				return
			}

			err = h.repo.CreateUser(req.Name, hashPwd)
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

func (h *handler) Login(c *gin.Context) {
	name, ok := getFromCtx(c, "username")
	if !ok {
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	token := hex.EncodeToString(key)

	h.rdb.Record(h.ctx, token, name, 80*time.Second)

	c.SetCookie("token", token, 80, "/", "localhost", false, true)
	c.JSON(http.StatusOK, "Login success!🫦")
}

func (h *handler) Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "molodec! 👍",
	})
}

func (h *handler) Logout(c *gin.Context) {
	token, ok := getFromCtx(c, "token")
	if !ok {
		return
	}

	h.rdb.Delete(h.ctx, token)

	c.SetCookie("token", "", -1, "/", "localhost", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "you've got rid of 🍪🗑️",
	})
}

func (h *handler) Delacc(c *gin.Context) {
	name, ok := getFromCtx(c, "username")
	if !ok {
		return
	}

	token, ok := getFromCtx(c, "token")
	if !ok {
		return
	}

	err := h.repo.DeleteUser(name)
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

	h.rdb.Delete(h.ctx, token)
	c.SetCookie("token", "", -1, "/", "localhost", false, true)

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
