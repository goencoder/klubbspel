#!/bin/bash

# Klubbspel Deployment Script
# This script helps deploy the application to Fly.io

set -e

echo "🚀 Klubbspel Deployment Script"
echo "================================"

# Check if flyctl is installed
if ! command -v flyctl &> /dev/null; then
    echo "❌ flyctl CLI not found. Please install it first:"
    echo "   curl -L https://fly.io/install.sh | sh"
    exit 1
fi

# Check if user is logged in
if ! flyctl auth whoami &> /dev/null; then
    echo "❌ Not logged in to Fly.io"
    echo "📝 Please login first: flyctl auth login"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Function to deploy backend
deploy_backend() {
    echo "🔧 Deploying Backend..."
    echo "======================"
    
    # Check if app exists, create if not
    if ! flyctl apps list | grep -q "klubbspel-backend"; then
        echo "📱 Creating backend app..."
        flyctl apps create klubbspel-backend --org personal
    fi
    
    echo "🏗️  Building and deploying backend..."
    flyctl deploy --config fly-backend.toml --app klubbspel-backend
    
    echo "✅ Backend deployed successfully!"
    echo "🌐 Backend URL: https://klubbspel-backend.fly.dev"
    echo ""
}

# Function to deploy frontend
deploy_frontend() {
    echo "🎨 Deploying Frontend..."
    echo "======================"
    
    # Check if app exists, create if not
    if ! flyctl apps list | grep -q "klubbspel-frontend"; then
        echo "📱 Creating frontend app..."
        flyctl apps create klubbspel-frontend --org personal
    fi
    
    echo "🏗️  Building and deploying frontend..."
    flyctl deploy --config fly-frontend.toml --app klubbspel-frontend
    
    echo "✅ Frontend deployed successfully!"
    echo "🌐 Frontend URL: https://klubbspel-frontend.fly.dev"
    echo ""
}

# Function to setup secrets
setup_secrets() {
    echo "🔐 Setting up secrets..."
    echo "======================"
    
    echo "⚠️  You need to manually set the following secrets:"
    echo ""
    echo "For Backend (MongoDB Atlas):"
    echo "  flyctl secrets set MONGO_URI='mongodb+srv://username:password@cluster.mongodb.net/pingis?retryWrites=true&w=majority' --app klubbspel-backend"
    echo "  flyctl secrets set MONGO_DB='pingis' --app klubbspel-backend"
    echo ""
    echo "For SendGrid (if using email features):"
    echo "  flyctl secrets set SENDGRID_API_KEY='your-sendgrid-api-key' --app klubbspel-backend"
    echo ""
    echo "💡 Replace the connection strings with your actual MongoDB Atlas credentials"
    echo ""
}

# Function to check deployment status
check_status() {
    echo "📊 Checking deployment status..."
    echo "==============================="
    
    echo "Backend status:"
    flyctl status --app klubbspel-backend || echo "❌ Backend not deployed yet"
    echo ""
    
    echo "Frontend status:"
    flyctl status --app klubbspel-frontend || echo "❌ Frontend not deployed yet"
    echo ""
}

# Main menu
case "${1:-}" in
    "backend")
        deploy_backend
        ;;
    "frontend")
        deploy_frontend
        ;;
    "full")
        deploy_backend
        deploy_frontend
        setup_secrets
        ;;
    "secrets")
        setup_secrets
        ;;
    "status")
        check_status
        ;;
    *)
        echo "Usage: $0 {backend|frontend|full|secrets|status}"
        echo ""
        echo "Commands:"
        echo "  backend   - Deploy only the backend"
        echo "  frontend  - Deploy only the frontend" 
        echo "  full      - Deploy both backend and frontend"
        echo "  secrets   - Show how to setup secrets"
        echo "  status    - Check deployment status"
        echo ""
        echo "Example full deployment:"
        echo "  $0 full"
        exit 1
        ;;
esac

echo "🎉 Deployment script completed!"
