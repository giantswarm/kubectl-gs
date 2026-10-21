package login

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWriteLoginPage(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		err        error
		wantStatus int
		want       []string
		notWant    []string
	}{
		{
			name:       "signed in",
			query:      "code=c&state=s",
			wantStatus: http.StatusOK,
			want:       []string{"<h1>Signed in</h1>", "kubectl gs is now signed in", "return to the terminal"},
		},
		{
			name:       "provider refuses",
			query:      "error=access_denied&error_description=%3Cb%3Eno%3C%2Fb%3E&state=s",
			err:        errors.New("no code"),
			wantStatus: http.StatusBadRequest,
			want:       []string{"<h1>Sign-in failed</h1>", "The sign-in was denied or failed", "<code>access_denied</code>", "The terminal shows the details"},
			notWant:    []string{"<b>no</b>"},
		},
		{
			name:       "provider error that is not a registered code",
			query:      "error=your_account_is_locked_call_555&state=s",
			err:        errors.New("no code"),
			wantStatus: http.StatusBadRequest,
			want:       []string{"The sign-in was denied or failed"},
			notWant:    []string{"your_account_is_locked"},
		},
		{
			name:       "sign-in not completed",
			query:      "code=c&state=s",
			err:        errors.New("token exchange failed: internal detail"),
			wantStatus: http.StatusBadRequest,
			want:       []string{"<h1>Sign-in failed</h1>", "The sign-in could not be completed"},
			notWant:    []string{"internal detail"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()

			writeLoginPage(rr, query, tc.err)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if !strings.Contains(rr.Header().Get("Content-Security-Policy"), "default-src 'none'") {
				t.Errorf("expected a strict Content-Security-Policy, got %q", rr.Header().Get("Content-Security-Policy"))
			}
			body := rr.Body.String()
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q: %s", want, body)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(body, notWant) {
					t.Errorf("body contains %q", notWant)
				}
			}
		})
	}
}
