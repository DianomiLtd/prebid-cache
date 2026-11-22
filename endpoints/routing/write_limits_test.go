package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/stretchr/testify/require"
)

func TestWriteBodyLimit(t *testing.T) {
	const body = `{"puts":[{"type":"xml","key":"sample","value":"<VAST/>"}]}`
	cfg := config.Configuration{
		Routes: config.Routes{AllowPublicWrite: true},
		RequestLimits: config.RequestLimits{
			AllowSettingKeys: true,
			MaxNumValues:     1,
			MaxBodyBytes:     int64(len(body)),
		},
	}
	for name, router := range map[string]func(config.Configuration, backends.Backend, *metrics.Metrics) http.Handler{
		"public": NewPublicHandler, "admin": NewAdminHandler,
	} {
		for _, tc := range []struct {
			name    string
			payload string
			length  int64
			status  int
		}{
			{"exact", body, int64(len(body)), http.StatusOK},
			{"exact streamed", body, -1, http.StatusOK},
			{"over", body + " ", int64(len(body) + 1), http.StatusRequestEntityTooLarge},
			{"over streamed", body + " ", -1, http.StatusRequestEntityTooLarge},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				store := backends.NewMemoryBackend()
				handler := router(cfg, store, &metrics.Metrics{})
				req := httptest.NewRequest(http.MethodPost, "/cache", strings.NewReader(tc.payload))
				req.ContentLength = tc.length
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				require.Equal(t, tc.status, w.Code)
				value, err := store.Get(context.Background(), "sample")
				if tc.status == http.StatusOK {
					require.NoError(t, err)
					require.Equal(t, "xml<VAST/>", value)
				} else {
					require.Error(t, err)
					require.Equal(t, "Request body exceeds the configured limit.\n", w.Body.String())
				}
			})
		}
	}
}
