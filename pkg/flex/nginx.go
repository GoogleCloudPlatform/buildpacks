// Package flex contains functions to configure Flex applications.
package flex

import (
	"text/template"
)

// NginxConfig is the config for the Flex nginx template.
type NginxConfig struct {
	MimeTypesPath,
	NginxServerConfPath,
	NginxConfHTTPInclude string
}

// NginxConfTemplate is template for Flex Nginx config.
var NginxConfTemplate = template.Must(template.New("nginx").Parse(`
daemon off;
worker_processes auto;
error_log /dev/stderr info;

events {
        worker_connections 1024;
        use epoll;
}

http {
        tcp_nopush on;
        tcp_nodelay on;
        keepalive_timeout 24h;
        types_hash_max_size 2048;
        server_tokens off;
        client_max_body_size 32m;

        # Set connect timeout to max to correspond with long request timeout.
        proxy_connect_timeout 75s;
        proxy_read_timeout 24h;
        proxy_request_buffering off;
        proxy_buffering off;
        proxy_buffer_size 8k;

        default_type application/octet-stream;
        access_log /dev/null;

        gzip on;
        gzip_vary on;
        gzip_comp_level 1;
        gzip_proxied any;
        gzip_types
                text/plain
                text/css
                application/json
                application/javascript
                application/x-javascript
                application/wasm
                text/xml
                application/xml
                application/xml+rss
                text/javascript;

        include {{.MimeTypesPath}};

        include {{.NginxServerConfPath}};

        {{if .NginxConfHTTPInclude}}
        include {{.NginxConfHTTPInclude}};
        {{end}}
}`))
