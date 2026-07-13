package admin

import (
	"net/mail"
	"strings"
)

const (
	adminPasswordMinLength = 8
	adminPasswordMaxLength = 72
)

func normalizeAdminEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateAdminEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func validateAdminPassword(password string) bool {
	length := len(password)
	return length >= adminPasswordMinLength && length <= adminPasswordMaxLength
}
