package middleware

import (
	"github.com/gemfast/server/internal/db"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OtelEnrich attaches investigation-useful attributes to the active server span
// after the handler runs: authenticated user identity (when present) and an
// error.category derived from the final HTTP status.
//
// Must be installed AFTER otelgin (so a span exists) but ordering relative to
// auth middleware does not matter — c.Next() drains the chain before we read
// identity or status.
func OtelEnrich() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		span := trace.SpanFromContext(c.Request.Context())
		if !span.SpanContext().IsValid() {
			return
		}

		if v, ok := c.Get(IdentityKey); ok {
			if u, ok := v.(*db.User); ok && u != nil {
				attrs := []attribute.KeyValue{
					attribute.String("user.id", u.Username),
					attribute.String("user.role", u.Role),
				}
				if u.Type != "" {
					attrs = append(attrs, attribute.String("user.auth_type", u.Type))
				}
				span.SetAttributes(attrs...)
			}
		}

		if cat := categorizeStatus(c.Writer.Status()); cat != "" {
			span.SetAttributes(attribute.String("error.category", cat))
		}
	}
}

func categorizeStatus(status int) string {
	switch {
	case status == 400:
		return "validation"
	case status == 401, status == 403:
		return "auth"
	case status == 404:
		return "not_found"
	case status == 405:
		return "policy"
	case status >= 500:
		return "internal"
	default:
		return ""
	}
}
