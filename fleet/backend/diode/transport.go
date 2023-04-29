package diode

import (
	"context"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/orb-community/orb/pkg/types"
)

func MakeDiodeHandler(tracer opentracing.Tracer, pkt diodeBackend, opts []kithttp.ServerOption, r *bone.Mux) {

	r.Get("/agents/backends/diode/handlers", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_agent_backend_handler")(viewAgentBackendHandlerEndpoint(pkt)),
		decodeBackendView,
		types.EncodeResponse,
		opts...))
	r.Get("/agents/backends/diode/inputs", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_agent_backend_input")(viewAgentBackendInputEndpoint(pkt)),
		decodeBackendView,
		types.EncodeResponse,
		opts...))
	r.Get("/agents/backends/diode/taps", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_agent_backend_taps")(viewAgentBackendTapsEndpoint(pkt)),
		decodeBackendView,
		types.EncodeResponse,
		opts...))
}

func decodeBackendView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewResourceReq{
		token: parseJwt(r),
	}
	return req, nil
}

func parseJwt(r *http.Request) (token string) {
	if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		token = r.Header.Get("Authorization")[7:]
	}
	return
}
