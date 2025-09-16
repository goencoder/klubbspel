#!/bin/bash

# Klubbspel Deployment Script - Step by Step
# Using your actual MongoDB Atlas cluster

set -e

echo "🚀 Klubbspel Deployment - Step by Step"
echo "======================================="

# Your MongoDB connection details
MONGO_BASE_URL="mongodb+srv://klubbspel:<db_password>@klubbspel.zhdqxae.mongodb.net/"
DB_NAME="pingis"

echo "📋 Pre-deployment checklist:"
echo "1. ✅ Fly.io CLI installed"
echo "2. ✅ Logged in as goran@goencoder.se"
echo "3. ✅ MongoDB Atlas cluster ready at klubbspel.zhdqxae.mongodb.net"
echo "4. ⚠️  Need database password for deployment"
echo ""

# Function to deploy with secrets
deploy_with_secrets() {
    echo "🔐 Please provide your MongoDB Atlas database password:"
    read -s MONGO_PASSWORD
    
    # Construct full MongoDB URI
    MONGO_URI="${MONGO_BASE_URL/<db_password>/$MONGO_PASSWORD}${DB_NAME}?retryWrites=true&w=majority"
    
    echo ""
    echo "🔧 Step 1: Creating and deploying backend..."
    
    # Create backend app if it doesn't exist
    if ! flyctl apps list | grep -q "klubbspel-backend"; then
        echo "📱 Creating backend app..."
        flyctl apps create klubbspel-backend --org personal
    else
        echo "📱 Backend app already exists"
    fi
    
    echo "🏗️  Deploying backend..."
    flyctl deploy --config fly-backend.toml --app klubbspel-backend
    
    echo "🔐 Setting up backend secrets..."
    flyctl secrets set MONGO_URI="$MONGO_URI" --app klubbspel-backend
    flyctl secrets set MONGO_DB="$DB_NAME" --app klubbspel-backend
    
    echo "✅ Backend deployed and configured!"
    echo "🌐 Backend URL: https://klubbspel-backend.fly.dev"
    echo ""
    
    # Test backend health
    echo "🩺 Testing backend health..."
    sleep 10  # Give it time to start
    if curl -f https://klubbspel-backend.fly.dev/healthz 2>/dev/null; then
        echo "✅ Backend is healthy!"
    else
        echo "⚠️  Backend health check failed - check logs with: flyctl logs --app klubbspel-backend"
    fi
    echo ""
    
    echo "🎨 Step 2: Creating and deploying frontend..."
    
    # Create frontend app if it doesn't exist
    if ! flyctl apps list | grep -q "klubbspel-frontend"; then
        echo "📱 Creating frontend app..."
        flyctl apps create klubbspel-frontend --org personal
    else
        echo "📱 Frontend app already exists"
    fi
    
    echo "🏗️  Deploying frontend..."
    flyctl deploy --config fly-frontend.toml --app klubbspel-frontend
    
    echo "✅ Frontend deployed!"
    echo "🌐 Frontend URL: https://klubbspel-frontend.fly.dev"
    echo ""
    
    echo "🎉 Deployment completed successfully!"
    echo ""
    echo "📊 Your Klubbspel application is now live:"
    echo "  Frontend: https://klubbspel-frontend.fly.dev"
    echo "  Backend:  https://klubbspel-backend.fly.dev"
    echo "  Health:   https://klubbspel-backend.fly.dev/healthz"
    echo ""
    echo "🔍 Useful commands:"
    echo "  flyctl logs --app klubbspel-backend    # Backend logs"
    echo "  flyctl logs --app klubbspel-frontend   # Frontend logs"
    echo "  flyctl status --app klubbspel-backend  # Backend status"
    echo "  flyctl status --app klubbspel-frontend # Frontend status"
}

# Function to just show the manual steps
show_manual_steps() {
    echo "📋 Manual Deployment Steps:"
    echo "=========================="
    echo ""
    echo "1. Deploy Backend:"
    echo "   flyctl apps create klubbspel-backend --org personal"
    echo "   flyctl deploy --config fly-backend.toml --app klubbspel-backend"
    echo ""
    echo "2. Set Backend Secrets (replace YOUR_PASSWORD):"
    echo "   flyctl secrets set MONGO_URI='mongodb+srv://klubbspel:YOUR_PASSWORD@klubbspel.zhdqxae.mongodb.net/pingis?retryWrites=true&w=majority' --app klubbspel-backend"
    echo "   flyctl secrets set MONGO_DB='pingis' --app klubbspel-backend"
    echo ""
    echo "3. Deploy Frontend:"
    echo "   flyctl apps create klubbspel-frontend --org personal"
    echo "   flyctl deploy --config fly-frontend.toml --app klubbspel-frontend"
    echo ""
    echo "4. Test:"
    echo "   curl https://klubbspel-backend.fly.dev/healthz"
    echo "   open https://klubbspel-frontend.fly.dev"
}

# Main menu
case "${1:-interactive}" in
    "auto")
        deploy_with_secrets
        ;;
    "manual")
        show_manual_steps
        ;;
    "interactive"|*)
        echo "Choose deployment method:"
        echo "1. Interactive deployment (recommended)"
        echo "2. Show manual steps"
        echo ""
        read -p "Enter choice [1-2]: " choice
        case $choice in
            1) deploy_with_secrets ;;
            2) show_manual_steps ;;
            *) echo "Invalid choice"; exit 1 ;;
        esac
        ;;
esac
