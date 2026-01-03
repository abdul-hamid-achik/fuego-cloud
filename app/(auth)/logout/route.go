package logout

import (
	"net/http"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

// Post handles logout by clearing the access_token cookie
// POST /logout
func Post(c *fuego.Context) error {
	// Clear the access_token cookie
	http.SetCookie(c.Response, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete the cookie
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(200, map[string]string{"message": "logged out successfully"})
}

// Get handles logout via GET (for browser redirects)
// GET /logout
func Get(c *fuego.Context) error {
	// Clear the access_token cookie
	http.SetCookie(c.Response, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to login page
	c.Response.Header().Set("Location", "/login")
	return c.JSON(302, nil)
}
