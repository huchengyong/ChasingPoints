package wechat

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
)

// MessagePushPath is the public WeChat message-push verification callback path.
const MessagePushPath = "/api/wechat/message-push"

func MessagePushHandler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}

		query := r.URL.Query()
		signature := query.Get("signature")
		timestamp := query.Get("timestamp")
		nonce := query.Get("nonce")
		echostr := query.Get("echostr")
		if signature == "" || timestamp == "" || nonce == "" || echostr == "" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		parts := []string{token, timestamp, nonce}
		sort.Strings(parts)
		sum := sha1.Sum([]byte(strings.Join(parts, "")))
		expectedSignature := hex.EncodeToString(sum[:])
		if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(signature)) != 1 {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(echostr))
	}
}
