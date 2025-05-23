package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("adgihioasxbfjkcbAEWIOFGHBIOHasegfWEAWEgWEARx")

type Claims struct {
	UserId string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userId string) (string, error) {
	// 生成token
	claims := Claims{
		userId,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 过期时间24小时
			IssuedAt:  jwt.NewNumericDate(time.Now()),                     // 签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                     // 生效时间
		},
	}
	// 使用HS256签名算法
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(jwtKey))

	return s, err
}

func ParseToken(tokenstring string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenstring, &Claims{}, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})

	if claims, ok := t.Claims.(*Claims); ok && t.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}
