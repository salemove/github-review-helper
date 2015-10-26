package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
)

func checkAuthentication(body []byte, r *http.Request, secret string) *ErrorResponse {
	signature := r.Header.Get("X-Hub-Signature")
	if signature == "" {
		return &ErrorResponse{nil, http.StatusUnauthorized, "Please provide a X-Hub-Signature"}
	}
	hasSecret, err := hasSecret(body, signature, secret)
	if err != nil {
		return &ErrorResponse{err, http.StatusInternalServerError, "Failed to check the signature"}
	} else if !hasSecret {
		return &ErrorResponse{nil, http.StatusForbidden, "Bad X-Hub-Signature"}
	}
	return nil
}

func hasSecret(message []byte, signature, key string) (bool, error) {
	var messageMACString string
	fmt.Sscanf(signature, "sha1=%s", &messageMACString)
	messageMAC, err := hex.DecodeString(messageMACString)
	if err != nil {
		return false, err
	}

	mac := hmac.New(sha1.New, []byte(key))
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC), nil
}
