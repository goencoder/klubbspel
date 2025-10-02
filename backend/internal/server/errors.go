package server

import (
	"context"
	"net/http"

	runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Localized error handler for grpc-gateway
func LocalizedErrorHandler() runtime.ErrorHandlerFunc {
	return func(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
		st, _ := status.FromError(err)
		code := httpStatus(st.Code())
		w.WriteHeader(code)
		_ = m.NewEncoder(w).Encode(map[string]any{
			"code":    st.Message(), // we'll encode stable codes in Status message
			"message": st.Message(), // replace with i18n lookup if you prefer
		})
	}
}

func httpStatus(c codes.Code) int { return runtime.HTTPStatusFromCode(c) }
