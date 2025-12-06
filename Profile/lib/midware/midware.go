package midware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wladim1r/profile/lib/getenv"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func verifyToken(c *gin.Context, token string, refresh bool) error {
	parser := jwt.NewParser()

	if refresh {
		parser = jwt.NewParser(jwt.WithoutClaimsValidation())
	}

	jwtToken, err := parser.Parse(token, func(t *jwt.Token) (any, error) {
		return []byte(getenv.GetString("SECRET_KEY", "Tralalelo tralala")), nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}
	if !jwtToken.Valid {
		return fmt.Errorf("token is not valid")
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("could not parse jwtToken into MapClaims struct")
	}

	if !refresh {
		expRaw, ok := claims["exp"]
		if !ok {
			return fmt.Errorf("field 'expiration' did not found")
		}
		exp64, ok := expRaw.(float64)
		if !ok {
			return fmt.Errorf("failed to parse expRaw into int64")
		}
		exp := int64(exp64)

		if time.Now().Unix() > exp {
			return fmt.Errorf("token life time has expired")
		}
	}

	userIDRaw, ok := claims["sub"]
	if !ok {
		return fmt.Errorf("field 'userID' did not found")
	}
	userID, ok := userIDRaw.(float64)
	if !ok {
		return fmt.Errorf("failed to parse 'userIDRaw' into float64")
	}

	c.Set("user_id", userID)
	return nil
}

func CheckAuth(refresh bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeadVal := c.GetHeader("Authorization")
		if authHeadVal == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeadVal, " ")
		if len(bearerToken) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		if err := verifyToken(c, bearerToken[1], refresh); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
