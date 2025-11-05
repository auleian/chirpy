package auth

import (
	"log"
	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hashed, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Fatal(err)
	}
	return string(hashed), nil
}

func CheckPasswordHash(password, hash string) (bool, error){
	match, err := argon2id.ComparePasswordAndHash("pa$$word", hash)
	if err != nil {
		log.Fatal(err)
	}

	return match, nil
}