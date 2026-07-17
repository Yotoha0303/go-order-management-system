package observability

type Metrics struct {
	HTTP     *HTTPMetrics
	Business *BusinessMetrics
}

func NewMetrics() *Metrics {
	return &Metrics{
		HTTP:     NewHTTPMetrics(),
		Business: NewBusinessMetrics(),
	}
}

func (m *Metrics) RenderPrometheus() string {
	if m == nil {
		return ""
	}
	return m.HTTP.RenderPrometheus() + m.Business.RenderPrometheus()
}
