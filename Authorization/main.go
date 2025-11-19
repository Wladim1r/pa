package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Request struct {
	Name     string `json:"name"     binding:"required"`
	Password string `json:"password" binding:"required"`
}

var tokensDB = make(map[string]string)

type UsersDB struct {
	db *sql.DB
}

func (db *UsersDB) CreateUser(name string, password []byte) error {
	_, err := db.db.Exec("INSERT INTO users (name, password) VALUES (?, ?)", name, password)
	if err != nil {
		return err
	}

	return nil
}

func (db *UsersDB) DeleteUser(name string) error {
	_, err := db.db.Exec("DELETE FROM users WHERE name = ?", name)
	if err != nil {
		return err
	}

	return nil
}

func (db *UsersDB) SelectPwdByName(name string) (string, error) {
	row := db.db.QueryRow("SELECT password FROM users WHERE name = ?", name)

	var password string
	if err := row.Scan(&password); err != nil {
		return "", err
	}

	return password, nil
}

func NewUserDB() *UsersDB {
	db, err := sql.Open("sqlite", "./users.db")
	if err != nil {
		slog.Error("Could not open db sqlite", "error", err)
		os.Exit(1)
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, password TEXT)"); err != nil {
		slog.Error("Could not create table", "error", err)
		os.Exit(1)
	}

	return &UsersDB{
		db: db,
	}
}

func CheckCookie(db *UsersDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"no cookie": err.Error(),
			})
			c.Abort()
			return
		}

		name, ok := tokensDB[token]
		if !ok {
			c.JSON(http.StatusConflict, gin.H{
				"error": "invalid cookie",
			})
			c.Abort()
			return
		}

		c.Set("token", token)
		c.Set("username", name)
		c.Next()
	}
}

func CheckUserExists(db *UsersDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		password, err := db.SelectPwdByName(req.Name)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "user does not registered ❌",
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "db error: " + err.Error(),
			})
			c.Abort()
			return
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

func HashPwd(db *UsersDB, pwd []byte, name string) ([]byte, error) {
	hashPas, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return hashPas, nil
}

func main() {
	db := NewUserDB()
	defer db.db.Close()

	r := gin.Default()

	r.POST("/register", func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		row := db.db.QueryRow("SELECT id FROM users WHERE name = ?", req.Name)

		var id int
		err := row.Scan(&id)

		if err != nil {
			if err == sql.ErrNoRows {
				hashPwd, err := HashPwd(db, []byte(req.Password), req.Name)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"Could not hash password": err.Error(),
					})
					return
				}

				err = db.CreateUser(req.Name, hashPwd)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"Could not create user": err.Error(),
					})
					return
				}

				c.JSON(http.StatusCreated, gin.H{
					"message": "user successful created 🎊🤩",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "db error: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusConflict, gin.H{
			"message": "user already exsited 💩",
		})
	})

	r.POST("/login", CheckUserExists(db), func(c *gin.Context) {
		name, ok := GetFromCtx(c, "username")
		if !ok {
			return
		}

		key := make([]byte, 32)
		rand.Read(key)
		token := hex.EncodeToString(key)

		tokensDB[token] = name

		c.SetCookie("token", token, 80, "/", "localhost", false, true)
		c.JSON(http.StatusOK, "Login success!🫦")
	})

	logined := r.Group("/auth")
	logined.Use(CheckCookie(db))
	{
		logined.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "molodec! 👍",
			})
		})

		logined.POST("/logout", func(c *gin.Context) {
			token, ok := GetFromCtx(c, "token")
			if !ok {
				return
			}

			delete(tokensDB, token)

			c.SetCookie("token", "", -1, "/", "localhost", false, true)
			c.JSON(http.StatusOK, gin.H{
				"message": "you've got rid of 🍪🗑️",
			})
		})

		logined.POST("/delacc", func(c *gin.Context) {
			name, ok := GetFromCtx(c, "username")
			if !ok {
				return
			}

			token, ok := GetFromCtx(c, "token")
			if !ok {
				return
			}

			err := db.DeleteUser(name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "❌🗑️ Could not delete user: " + err.Error(),
				})
				return
			}

			delete(tokensDB, token)
			c.SetCookie("token", "", -1, "/", "localhost", false, true)

			c.JSON(http.StatusOK, gin.H{
				"message": "👍 user have successful deleted from DB",
			})
		})
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	server.ListenAndServe()
}

func GetFromCtx(c *gin.Context, key string) (string, bool) {
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
