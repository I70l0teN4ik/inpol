package pkg

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// todo: find out secret from inpol devs :joke:
func generateToken(conf Config) string {
	now := time.Now()
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         conf.UserID,
		"unique_name": conf.UserID,
		"jti":         uuid.NewString(),
		"iat":         now.Unix(),
		"displayName": conf.Email,
		"mfa":         "2FA",
		"nbf":         now.Unix(),
		"exp":         now.Add(time.Minute * 10).Unix(),
		"iss":         "inpol-direct",
		"aud":         "inpol-direct",
	})
	tokenString, _ := validToken.SignedString([]byte(conf.InpolSecret))

	return tokenString
}

func generateMFA(conf Config) string {
	now := time.Now()
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         conf.UserID,
		"unique_name": conf.UserID,
		"jti":         uuid.NewString(),
		"iat":         now.Unix(),
		"prp":         2,
		"nbf":         now.Unix(),
		"exp":         now.Add(time.Second * 30).Unix(),
		"iss":         "inpol-direct",
		"aud":         "inpol-direct-2fa",
	})
	tokenString, _ := validToken.SignedString([]byte(conf.InpolSecret))

	return tokenString
}
