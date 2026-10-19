package http

import "strings"

// Reserved usernames: mencegah tabrakan dengan route statis FE
// (/{username} vs /portfolios, /search, dst) + nama sensitif sistem.
var reservedUsernames = map[string]bool{
	"admin": true, "administrator": true, "root": true, "system": true, "support": true,
	"login": true, "logout": true, "register": true, "signup": true, "auth": true,
	"me": true, "api": true, "profile": true, "settings": true,
	"portfolios": true, "users": true, "user": true, "search": true,
	"notifications": true, "changelog": true, "feedback": true,
	"feed": true, "leaderboards": true, "series": true, "tags": true,
	"static": true, "assets": true, "public": true, "help": true,
	"about": true, "grafikarsa": true, "smkn4": true, "moderator": true,
}

func usernameReserved(u string) bool {
	return reservedUsernames[strings.ToLower(strings.TrimSpace(u))]
}
