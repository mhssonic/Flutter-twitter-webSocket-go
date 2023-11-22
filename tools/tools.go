package tools

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"strconv"
)

const key = "ba esm ramz pashmak"

func JwtDecode(strToken string) (int, error) {
	var user jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(strToken, &user, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		return 0, err //TODO check it
	}
	if token.Valid {
		return strconv.Atoi(user.ID)
	} else {
		return 0, errors.New("not valid")
	}
}

func JenkinsHash(a int, b int, sort bool) int {
	if sort && a < b {
		c := a
		a = b
		b = c
	}

	hash := int32(0)

	hash += int32(a)
	hash += hash << 10
	hash ^= int32(uint32(hash) >> 6)

	hash += int32(b)
	hash += hash << 10
	hash ^= int32(uint32(hash) >> 6)

	hash += hash << 3
	hash ^= int32(uint32(hash) >> 11)
	hash += hash << 15

	return int(hash)
}
