package svg

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"html/template"

	"github.com/zserge/metric"
)

var (
	page = template.Must(template.New("").
		Funcs(template.FuncMap{"path": path, "duration": duration}).
		Parse(`
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 20">
{{ if eq (index (index .samples 0) "type") "c" }}
	{{ range (path .samples "count") }}<path d={{ . }} />{{end}}
{{ else if eq (index (index .samples 0) "type") "g" }}
	{{ range (path .samples "min" "max" "mean" ) }}<path d={{ . }} />{{end}}
{{ else if eq (index (index .samples 0) "type") "h" }}
	{{ range (path .samples "p50" "p90" "p99") }}<path d={{ . }} />{{end}}
{{ end }}
</svg>
`))
)

func path(samples []interface{}, keys ...string) []string {
	var min, max float64
	paths := make([]string, len(keys), len(keys))
	for i := 0; i < len(samples); i++ {
		s := samples[i].(map[string]interface{})
		for _, k := range keys {
			x := s[k].(float64)
			if i == 0 || x < min {
				min = x
			}
			if i == 0 || x > max {
				max = x
			}
		}
	}
	for i := 0; i < len(samples); i++ {
		s := samples[i].(map[string]interface{})
		for j, k := range keys {
			v := s[k].(float64)
			x := float64(i+1) / float64(len(samples))
			y := (v - min) / (max - min)
			if max == min {
				y = 0
			}
			if i == 0 {
				paths[j] = fmt.Sprintf("M%f %f", 0.0, (1-y)*18+1)
			}
			paths[j] += fmt.Sprintf(" L%f %f", x*100, (1-y)*18+1)
		}
	}
	return paths
}

func duration(samples []interface{}, n float64) string {
	n = n * float64(len(samples))
	if n < 60 {
		return fmt.Sprintf("%d sec", int(n))
	} else if n < 60*60 {
		return fmt.Sprintf("%d min", int(n/60))
	} else if n < 24*60*60 {
		return fmt.Sprintf("%d hrs", int(n/60/60))
	}
	return fmt.Sprintf("%d days", int(n/24/60/60))
}

func Fprint(w io.Writer, snapshot func() map[string]metric.Metric) (err error) {
	type h map[string]interface{}
	metrics := []h{}
	for name, metric := range snapshot() {
		m := h{}
		b, _ := json.Marshal(metric)
		json.Unmarshal(b, &m)
		m["name"] = name
		metrics = append(metrics, m)
	}
	sort.Slice(metrics, func(i, j int) bool {
		n1 := metrics[i]["name"].(string)
		n2 := metrics[j]["name"].(string)
		return strings.Compare(n1, n2) < 0
	})
	for _, m := range metrics {
		err = page.Execute(w, m)
		if err != nil {
			return
		}
	}
	return
}
