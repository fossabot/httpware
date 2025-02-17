package tripperware_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gol4ng/httpware"
	"github.com/gol4ng/httpware/correlation_id"
	"github.com/gol4ng/httpware/mocks"
	"github.com/gol4ng/httpware/tripperware"
)

func TestMain(m *testing.M){
	correlation_id.DefaultIdGenerator = correlation_id.NewRandomIdGenerator(
		rand.New(correlation_id.NewLockedSource(rand.NewSource(1))),
	)
	os.Exit(m.Run())
}

func TestCorrelationId(t *testing.T) {
	roundTripperMock := &mocks.RoundTripper{}
	req := httptest.NewRequest(http.MethodGet, "http://fake-addr", nil)
	resp := &http.Response{
		Status:        "OK",
		StatusCode:    http.StatusOK,
		ContentLength: 30,
	}

	roundTripperMock.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Run(func(args mock.Arguments) {
		innerReq := args.Get(0).(*http.Request)
		assert.Len(t, innerReq.Header.Get(correlation_id.HeaderName), 10)
		assert.Equal(t, req.Header.Get(correlation_id.HeaderName), innerReq.Header.Get(correlation_id.HeaderName))
	})

	resp2, err := tripperware.CorrelationId(correlation_id.NewConfig())(roundTripperMock).RoundTrip(req)
	assert.Nil(t, err)
	assert.Equal(t, resp, resp2)
	assert.Equal(t, "p1LGIehp1s", req.Header.Get(correlation_id.HeaderName))
}

func TestCorrelationId_AlreadyInContext(t *testing.T) {
	config := correlation_id.NewConfig()
	roundTripperMock := &mocks.RoundTripper{}
	req := httptest.NewRequest(http.MethodGet, "http://fake-addr", nil)
	req = req.WithContext(context.WithValue(req.Context(), config.HeaderName, "my_already_exist_correlation_id"))

	resp := &http.Response{
		Status:        "OK",
		StatusCode:    http.StatusOK,
		ContentLength: 30,
	}

	roundTripperMock.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Run(func(args mock.Arguments) {
		innerReq := args.Get(0).(*http.Request)
		assert.Equal(t, req, innerReq)
		assert.Len(t, innerReq.Header.Get(config.HeaderName), 31)
		assert.Equal(t, req.Header.Get(config.HeaderName), innerReq.Header.Get(config.HeaderName))
	})

	resp2, err := tripperware.CorrelationId(config)(roundTripperMock).RoundTrip(req)
	assert.Nil(t, err)
	assert.Equal(t, resp, resp2)
	assert.Equal(t, "my_already_exist_correlation_id", req.Header.Get(config.HeaderName))
}

func TestCorrelationIdCustom(t *testing.T) {
	roundTripperMock := &mocks.RoundTripper{}
	req := httptest.NewRequest(http.MethodGet, "http://fake-addr", nil)
	resp := &http.Response{
		Status:        "OK",
		StatusCode:    http.StatusOK,
		ContentLength: 30,
	}

	roundTripperMock.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Run(func(args mock.Arguments) {
		innerReq := args.Get(0).(*http.Request)
		assert.Equal(t, "my_fake_correlation", innerReq.Header.Get(correlation_id.HeaderName))
	})

	config := correlation_id.NewConfig()
	config.IdGenerator = func(request *http.Request) string {
		return "my_fake_correlation"
	}

	resp2, err := tripperware.CorrelationId(config)(roundTripperMock).RoundTrip(req)
	assert.Nil(t, err)
	assert.Equal(t, resp, resp2)
}

// =====================================================================================================================
// ========================================= EXAMPLES ==================================================================
// =====================================================================================================================

func ExampleCorrelationId() {
	port := ":5005"
	config := correlation_id.NewConfig()
	// you can override default header name
	config.HeaderName = "my-personal-header-name"
	// you can override default id generator
	config.IdGenerator = func(request *http.Request) string {
		return "my-generated-id"
	}

	// we recommend to use MiddlewareStack to simplify managing all wanted middleware
	// caution middleware order matter
	stack := httpware.TripperwareStack(
		tripperware.CorrelationId(config),
	)

	// create http client using the tripperwareStack as RoundTripper
	client := http.Client{
		Transport: stack,
	}

	// create a server in order to show it work
	srv := http.NewServeMux()
	srv.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("server receive request with request id:", request.Header.Get(config.HeaderName))
	})

	go func() {
		if err := http.ListenAndServe(port, srv); err != nil {
			panic(err)
		}
	}()

	_, _ = client.Get("http://localhost" + port + "/")

	// Output: server receive request with request id: my-generated-id
}
