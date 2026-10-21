package login

import (
	"net/http"
	"net/url"

	"github.com/giantswarm/pagekit"
)

// writeLoginPage shows the outcome of the browser sign-in. query is the
// identity provider's redirect to the local callback, and err the result of
// handling it: nil once the code has been exchanged for a token.
func writeLoginPage(w http.ResponseWriter, query url.Values, err error) {
	if err == nil {
		pagekit.Render(w, http.StatusOK, pagekit.Page{
			Title:   "Signed in with kubectl gs",
			Heading: "Signed in",
			Summary: "kubectl gs is now signed in",
			Message: "You can close this tab and return to the terminal.",
		})
		return
	}

	summary := "The sign-in could not be completed"
	errorCode := query.Get("error")
	if errorCode != "" {
		summary = "The sign-in was denied or failed"
	}
	pagekit.Render(w, http.StatusBadRequest, pagekit.Page{
		Title:   "Sign-in with kubectl gs failed",
		Heading: "Sign-in failed",
		Summary: summary,
		Message: "The terminal shows the details. You can close this tab and run the login again there.",
		// Anyone can send the browser to the local callback with an error of
		// their choosing, so only a registered code is shown.
		Detail: pagekit.OAuthErrorCode(errorCode),
	})
}
