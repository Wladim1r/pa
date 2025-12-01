// Package handlers
package handlers

import (
	"net/http"
	"time"

	"github.com/Wladim1r/profile/internal/models"
	"github.com/Wladim1r/profile/lib/getenv"
	"github.com/Wladim1r/proto-crypto/gen/protos/auth-portfile"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type client struct {
	conn auth.AuthClient
}

func NewClient(c auth.AuthClient) *client {
	return &client{conn: c}
}

func (cl *client) Registration(c *gin.Context) {
	var req models.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err := cl.conn.Register(
		c.Request.Context(),
		&auth.AuthRequest{Name: req.Name, Password: req.Password},
	)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(grpcCodeToHTTP(st.Code()), gin.H{
				"error": st.Message(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown gRPC error",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user successful created 🎊🤩",
	})
}

func (cl *client) Login(c *gin.Context) {
	var req models.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var header metadata.MD
	resp, err := cl.conn.Login(
		c.Request.Context(),
		&auth.AuthRequest{Name: req.Name, Password: req.Password},
		grpc.Header(&header),
	)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(grpcCodeToHTTP(st.Code()), gin.H{
				"error": st.Message(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown gRPC error",
			})
		}
		return
	}

	userID, ok := header["x-user-id"]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "there is no 'x-user-id' in the gRPC headers",
		})
		return
	}

	access, refresh := resp.GetAccess(), resp.GetRefresh()

	domain := getenv.GetString("COOKIE_DOMAIN", "localhost")
	ttl := int(getenv.GetTime("REFRESH_TTL", 3*time.Minute).Seconds())

	c.SetCookie("refreshToken", refresh, ttl, "/", domain, false, true)
	c.SetCookie("userID", userID[0], ttl, "/", domain, false, true)

	tStruct := struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}{
		Access:  access,
		Refresh: refresh,
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Login successful 👅🍭",
		"🌐 Here is your tokens": tStruct,
	})
}

func (cl *client) Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "molodec! 👍",
	})
}

func (cl *client) Refresh(c *gin.Context) {
	userID, err := c.Cookie("userID")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "there is no 'userID' in the cookies: " + err.Error(),
		})
		return
	}
	refresh, err := c.Cookie("refreshToken")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "there is no 'refreshToken' in the cookies: " + err.Error(),
		})
		return
	}

	ctx := metadata.NewOutgoingContext(c.Request.Context(), metadata.Pairs(
		"x-user-id", userID,
		"x-refresh-token", refresh,
	))

	resp, err := cl.conn.Refresh(ctx, &auth.Empty{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(grpcCodeToHTTP(st.Code()), gin.H{
				"error": st.Message(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown gRPC error",
			})
		}
		return
	}

	tStruct := struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}{
		Access:  resp.GetAccess(),
		Refresh: resp.GetRefresh(),
	}

	domain := getenv.GetString("COOKIE_DOMAIN", "localhost")
	ttl := int(getenv.GetTime("REFRESH_TTL", 3*time.Minute).Seconds())

	c.SetCookie("refreshToken", tStruct.Refresh, ttl, "/", domain, false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":  "tokens succesfully refreshed ♻️🤍",
		"tokens 💁": tStruct,
	})
}

func (cl *client) Logout(c *gin.Context) {
	refresh, err := c.Cookie("refreshToken")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "there is no 'refreshToken' in the cookies: " + err.Error(),
		})
		return
	}

	authHeader := c.GetHeader("Authorization")

	ctx := metadata.NewOutgoingContext(c.Request.Context(), metadata.Pairs(
		"x-refresh-token", refresh,
		"x-authorization-header", authHeader,
	))

	_, err = cl.conn.Logout(ctx, &auth.Empty{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(grpcCodeToHTTP(st.Code()), gin.H{
				"error": st.Message(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown gRPC error: " + err.Error(),
			})
		}
		return
	}

	domain := getenv.GetString("COOKIE_DOMAIN", "localhost")

	c.SetCookie("userID", "", -1, "/", domain, false, true)
	c.SetCookie("refreshToken", "", -1, "/", domain, false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "user succesfully logouted 👉🚪",
	})
}

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.Internal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
