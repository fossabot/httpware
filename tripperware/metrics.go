package tripperware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gol4ng/httpware"
	"github.com/gol4ng/httpware/metrics"
)

func Metrics(config *metrics.Config) httpware.Tripperware {
	return func(next http.RoundTripper) http.RoundTripper {
		return httpware.RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
			handlerName := config.HandlerNameExtractor(req)
			if !config.DisableMeasureInflight {
				config.Recorder.AddInflightRequests(req.Context(), handlerName, 1)
				defer config.Recorder.AddInflightRequests(req.Context(), handlerName, -1)
			}

			start := time.Now()
			defer func() {
				statusCode := http.StatusServiceUnavailable
				contentLength := int64(0)
				if resp != nil {
					statusCode = resp.StatusCode
					contentLength = resp.ContentLength
				}
				code := strconv.Itoa(statusCode)
				if !config.SplitStatus {
					code = fmt.Sprintf("%dxx", statusCode/100)
				}

				config.Recorder.ObserveHTTPRequestDuration(req.Context(), handlerName, time.Since(start), req.Method, code)

				if !config.DisableMeasureSize {
					config.Recorder.ObserveHTTPResponseSize(req.Context(), handlerName, contentLength, req.Method, code)
				}
			}()

			return next.RoundTrip(req)
		})
	}
}
