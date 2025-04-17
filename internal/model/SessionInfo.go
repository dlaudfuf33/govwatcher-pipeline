package model

import (
	"context"
	"net/http"
)

type SessionInfo struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	CSRFToken string
	Cookies   []*http.Cookie
}