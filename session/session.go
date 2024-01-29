// session/session.go

package session

import (
    "github.com/gorilla/sessions"
)

// Store is the session store
var Store = sessions.NewCookieStore([]byte("your-secret-key"))
