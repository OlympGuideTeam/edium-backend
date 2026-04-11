package metrics

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "Количество HTTP-запросов по сервису, методу, пути и статусу",
}, []string{"service", "method", "path", "status"})

func Middleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		httpRequests.WithLabelValues(
			service,
			c.Request.Method,
			c.FullPath(),
			strconv.Itoa(c.Writer.Status()),
		).Inc()
	}
}
