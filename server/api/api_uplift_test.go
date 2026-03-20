package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	ht "github.com/ogen-go/ogen/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type mockHandler struct{ mock.Mock }

func (m *mockHandler) HealthCheck(ctx context.Context) (*HealthCheckOK, error) {
	args := m.Called(ctx)
	var resp *HealthCheckOK
	if v := args.Get(0); v != nil {
		resp = v.(*HealthCheckOK)
	}
	return resp, args.Error(1)
}

func TestServerRoutesAndMethods(t *testing.T) {
	h := new(mockHandler)
	h.On("HealthCheck", mock.Anything).Return(&HealthCheckOK{Status: "ok"}, nil).Once()

	srv, err := NewServer(h)
	require.NoError(t, err)

	r, ok := srv.FindRoute(http.MethodGet, "/health")
	require.True(t, ok)
	assert.Equal(t, HealthCheckOperation, r.Name())
	assert.Equal(t, "/health", r.PathPattern())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	srv.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var body map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])

	h.AssertExpectations(t)
}

func TestServerMethodNotAllowedAndOptions(t *testing.T) {
	srv, err := NewServer(UnimplementedHandler{})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/health", nil))
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Equal(t, "GET", rr.Header().Get("Allow"))

	rr = httptest.NewRecorder()
	opt := httptest.NewRequest(http.MethodOptions, "/health", nil)
	srv.ServeHTTP(rr, opt)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, "GET", rr.Header().Get("Access-Control-Allow-Methods"))
}

func TestServerPrefixAndNotFound(t *testing.T) {
	h := new(mockHandler)
	h.On("HealthCheck", mock.Anything).Return(&HealthCheckOK{Status: "ok"}, nil).Once()

	notFoundHit := false
	srv, err := NewServer(h,
		WithPathPrefix("/api"),
		WithNotFound(func(w http.ResponseWriter, r *http.Request) {
			notFoundHit = true
			http.NotFound(w, r)
		}),
	)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.True(t, notFoundHit)

	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	assert.Equal(t, http.StatusOK, rr.Code)
	h.AssertExpectations(t)
}

func TestClientHealthCheckAndOverrideURL(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/health", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer backend.Close()

	client, err := NewClient(backend.URL)
	require.NoError(t, err)

	resp, err := client.HealthCheck(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.GetStatus())

	overrideURL, err := url.Parse(backend.URL)
	require.NoError(t, err)
	ctx := WithServerURL(context.Background(), overrideURL)
	resp, err = client.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestClientAndDecoderErrors(t *testing.T) {
	_, err := NewClient("://bad-url")
	require.Error(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer backend.Close()

	client, err := NewClient(backend.URL)
	require.NoError(t, err)
	_, err = client.HealthCheck(context.Background())
	require.Error(t, err)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"status":"ok","extra":1}`)),
	}
	_, err = decodeHealthCheckResponse(resp)
	require.Error(t, err)
}

func TestEncodingDecodingAndHelpers(t *testing.T) {
	var nilTarget *HealthCheckOK
	require.Error(t, nilTarget.Decode(nil))

	var okResp HealthCheckOK
	require.NoError(t, okResp.UnmarshalJSON([]byte(`{"status":"ok"}`)))
	assert.Equal(t, "ok", okResp.Status)

	data, err := okResp.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok"}`, string(data))

	err = okResp.UnmarshalJSON([]byte(`{"unknown":1}`))
	require.Error(t, err)

	err = okResp.UnmarshalJSON([]byte(`{}`))
	require.Error(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := contextWithLabeler(req.Context(), &Labeler{})
	l, found := LabelerFromContext(ctx)
	require.True(t, found)
	l.Add()
	_, found = LabelerFromContext(context.Background())
	assert.False(t, found)

	uh := UnimplementedHandler{}
	_, err = uh.HealthCheck(context.Background())
	require.ErrorIs(t, err, ht.ErrNotImplemented)

	u, parseErr := url.Parse("https://example.com///")
	require.NoError(t, parseErr)
	trimTrailingSlashes(u)
	assert.Equal(t, "", strings.TrimPrefix(u.Path, "/"))

	// cover option helpers
	scfg := newServerConfig(
		WithPathPrefix("/x"),
		WithNotFound(http.NotFound),
		WithMethodNotAllowed(func(http.ResponseWriter, *http.Request, string) {}),
		WithErrorHandler(func(context.Context, http.ResponseWriter, *http.Request, error) {}),
		WithMaxMultipartMemory(1024),
	)
	assert.Equal(t, "/x", scfg.Prefix)
	assert.EqualValues(t, 1024, scfg.MaxMultipartMemory)

	ccfg := newClientConfig(WithClient(http.DefaultClient), WithTracerProvider(noop.NewTracerProvider()))
	assert.NotNil(t, ccfg.Client)

	// custom MethodNotAllowed branch
	called := false
	bs := baseServer{cfg: serverConfig{MethodNotAllowed: func(w http.ResponseWriter, r *http.Request, allowed string) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}}}
	rr := httptest.NewRecorder()
	bs.notAllowed(rr, httptest.NewRequest(http.MethodPost, "/", nil), notAllowedParams{allowedMethods: "GET"})
	assert.True(t, called)
	assert.Equal(t, http.StatusAccepted, rr.Code)

	// encode failure branch
	badWriter := &failingWriter{header: http.Header{}}
	err = encodeHealthCheckResponse(&HealthCheckOK{Status: "ok"}, badWriter, noop.Span{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errWriteFailed))
}

type failingWriter struct{ header http.Header }

var errWriteFailed = errors.New("write failed")

func (f *failingWriter) Header() http.Header { return f.header }
func (f *failingWriter) WriteHeader(statusCode int) {}
func (f *failingWriter) Write(p []byte) (n int, err error) { return 0, errWriteFailed }
