package formatter

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"time"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

// HTMLEmitter buffers events and generates an HTML report on Close.
type HTMLEmitter struct {
	events []event.Event
	w      io.Writer
}

// NewHTMLEmitter creates an emitter that buffers events for HTML output.
func NewHTMLEmitter(w io.Writer) *HTMLEmitter {
	return &HTMLEmitter{
		events: make([]event.Event, 0),
		w:      w,
	}
}

// Emit buffers an event.
func (e *HTMLEmitter) Emit(ev event.Event) error {
	e.events = append(e.events, ev)
	return nil
}

// Close generates the HTML report and writes it to the configured writer.
func (e *HTMLEmitter) Close() error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"formatTime": func(ts int64) string {
			return time.Unix(0, ts).Format("15:04:05.000")
		},
		"formatData": func(data map[string]interface{}) template.HTML {
			if len(data) == 0 {
				return ""
			}
			result := ""
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				result += fmt.Sprintf("<div class=\"data-item\"><strong>%s:</strong> %s</div>",
					template.HTMLEscapeString(k),
					template.HTMLEscapeString(fmt.Sprintf("%v", data[k])))
			}
			return template.HTML(result)
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Events []event.Event
	}{
		Events: e.events,
	}

	return tmpl.Execute(e.w, data)
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Network Trace Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        h1 {
            color: #333;
        }
        .event {
            background: white;
            border-left: 4px solid #007bff;
            padding: 15px;
            margin: 10px 0;
            border-radius: 4px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .event-type {
            font-weight: bold;
            color: #007bff;
            font-size: 1.1em;
        }
        .event-time {
            color: #666;
            font-size: 0.9em;
        }
        .event-trace-id {
            color: #999;
            font-size: 0.85em;
            margin-left: 10px;
        }
        .event-data {
            margin-top: 10px;
            padding: 10px;
            background: #f9f9f9;
            border-radius: 3px;
        }
        .data-item {
            margin: 5px 0;
            font-family: monospace;
            font-size: 0.9em;
        }
        .dns_start, .dns_done { border-left-color: #28a745; }
        .tcp_connect_start, .tcp_connect_done { border-left-color: #17a2b8; }
        .tls_handshake_start, .tls_handshake_done { border-left-color: #ffc107; }
        .http_request_start, .http_response_done { border-left-color: #dc3545; }
        .udp_send, .udp_receive { border-left-color: #6f42c1; }
    </style>
</head>
<body>
    <h1>Network Trace Report</h1>
    {{range .Events}}
    <div class="event {{.Type}}">
        <div>
            <span class="event-type">{{.Type}}</span>
            <span class="event-time">{{formatTime .Timestamp}}</span>
            <span class="event-trace-id">trace: {{.TraceID}}</span>
        </div>
        {{if .Data}}
        <div class="event-data">
            {{formatData .Data}}
        </div>
        {{end}}
    </div>
    {{end}}
</body>
</html>
`
