#!/bin/sh
echo "📦 Generating runtime config.json..."
echo "{\"API_BASE_URL\": \"${VITE_API_BASE_URL:-http://localhost:8080}\"}" > /usr/share/nginx/html/config.json
echo "🚀 Starting nginx..."
exec nginx -g "daemon off;"
