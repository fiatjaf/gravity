package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
)

type Entry struct {
	Owner string `json:"owner" db:"owner"`
	Name  string `json:"name" db:"name"`
	CID   string `json:"cid" db:"cid"`
	Note  string `json:"note,omitempty" db:"note"`
	Ext   string `json:"ext,omitempty" db:"ext"`
}

func validateJWT(token, owner string, claimsToValidate map[string]interface{}) error {
	// we get a jwt we must validate
	var pemstr string
	err := pg.Get(&pemstr, "SELECT pk FROM users WHERE name = $1", owner)
	if err != nil {
		return err
	}

	block, _ := pem.Decode([]byte(pemstr))
	if block == nil || block.Type != "PUBLIC KEY" {
		return err
	}

	pk, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return err
	}

	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return pk, nil
	})
	if err != nil {
		return err
	}

	// all data should be inside the token
	if claims, ok := t.Claims.(jwt.MapClaims); !ok || !t.Valid {
		return errors.New("Invalid JWT claims")
	} else {
		for k, v := range claimsToValidate {
			if claims[k] != v {
				return fmt.Errorf("Mismatched claim: %s != %s", claims[k], v)
			}
		}
	}

	return nil
}

func selectWithMatch(match string) string {
	return `
        SELECT owner, name, cid, coalesce(note, '') AS note, coalesce(ext, '') AS ext
        FROM head ` +
		match +
		`ORDER BY updated_at DESC`
}
