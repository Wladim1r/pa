package handler

import (
	"errors"
	"net/http"

	"github.com/Wladim1r/profile/internal/api/profile/service"
	"github.com/Wladim1r/profile/internal/models"
	"github.com/Wladim1r/profile/lib/errs"
	"github.com/gin-gonic/gin"
)

type handler struct {
	us service.UsersService
	cs service.CoinsService
}

func NewHandler(us service.UsersService, cs service.CoinsService) *handler {
	return &handler{us: us, cs: cs}
}

func (h *handler) GetCoins(c *gin.Context) {
	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	coins, err := h.cs.GetCoins(userID)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "coins not found",
			})
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"there are your coins": coins,
	})
}

func (h *handler) AddCoin(c *gin.Context) {
	var req models.CoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid body request",
		})
		return
	}

	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	if err := h.cs.AddCoin(userID, req.Symbol, req.Quantity); err != nil {
		switch {
		case errors.Is(err, errs.ErrDuplicated):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "coin has successfully added",
	})
}

func (h *handler) UpdateCoin(c *gin.Context) {
	var req models.CoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid body request",
		})
		return
	}

	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	if err := h.cs.UpdateCoin(userID, req.Symbol, req.Quantity); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "choosed coin updated V",
	})
}

func (h *handler) DeleteCoin(c *gin.Context) {
	var req models.CoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid body request",
		})
		return
	}

	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	if err := h.cs.DeleteCoin(userID, req.Symbol); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "choosed coin updated V",
	})
}

// ------------------------------------------------------------------

func (h *handler) GetUserProfile(c *gin.Context) {
	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	user, err := h.us.GetUserProfileByUserID(userID)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user profile": *user,
	})
}

func (h *handler) DeleteUserProfile(c *gin.Context) {
	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	if err := h.us.DeleteUserProfileByUserID(userID); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user has deleted",
	})
}
