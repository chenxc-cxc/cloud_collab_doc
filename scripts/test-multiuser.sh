#!/bin/bash

# Multi-user simulation test script

set -e

echo "🧪 Multi-User Simulation Test"
echo ""

API_URL=${API_URL:-http://localhost:8080}
WS_URL=${WS_URL:-ws://localhost:8081}

# Test users
ALICE_ID="11111111-1111-1111-1111-111111111111"
BOB_ID="22222222-2222-2222-2222-222222222222"
CHARLIE_ID="33333333-3333-3333-3333-333333333333"

echo "📋 Testing API Endpoints..."
echo ""

# Health check
echo "1. Health check..."
curl -s "$API_URL/health" | jq .
echo ""

# Login as Alice
echo "2. Login as Alice..."
ALICE_TOKEN=$(curl -s -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"alice@example.com"}' | jq -r '.token')
echo "   Token: ${ALICE_TOKEN:0:20}..."
echo ""

# List documents
echo "3. List documents (as Alice)..."
curl -s "$API_URL/api/docs" \
    -H "Authorization: Bearer $ALICE_TOKEN" | jq .
echo ""

# Create a new document
echo "4. Create new document..."
DOC=$(curl -s -X POST "$API_URL/api/docs" \
    -H "Authorization: Bearer $ALICE_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"title":"Test Document"}')
DOC_ID=$(echo $DOC | jq -r '.id')
echo "   Created document: $DOC_ID"
echo ""

# Add Bob as editor
echo "5. Add Bob as editor..."
curl -s -X PUT "$API_URL/api/docs/$DOC_ID/permissions" \
    -H "Authorization: Bearer $ALICE_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"$BOB_ID\",\"role\":\"edit\"}"
echo "   Done"
echo ""

# Add Charlie as viewer
echo "6. Add Charlie as viewer..."
curl -s -X PUT "$API_URL/api/docs/$DOC_ID/permissions" \
    -H "Authorization: Bearer $ALICE_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"$CHARLIE_ID\",\"role\":\"view\"}"
echo "   Done"
echo ""

# List permissions
echo "7. List permissions..."
curl -s "$API_URL/api/docs/$DOC_ID/permissions" \
    -H "Authorization: Bearer $ALICE_TOKEN" | jq .
echo ""

# Add a comment
echo "8. Add comment..."
curl -s -X POST "$API_URL/api/docs/$DOC_ID/comments" \
    -H "Authorization: Bearer $ALICE_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"content":"This is a test comment"}' | jq .
echo ""

# Login as Bob
echo "9. Login as Bob..."
BOB_TOKEN=$(curl -s -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"bob@example.com"}' | jq -r '.token')
echo "   Token: ${BOB_TOKEN:0:20}..."
echo ""

# Bob lists documents
echo "10. List documents (as Bob)..."
curl -s "$API_URL/api/docs" \
    -H "Authorization: Bearer $BOB_TOKEN" | jq .
echo ""

echo "✅ API tests complete!"
echo ""
echo "📡 WebSocket URL for testing:"
echo "   $WS_URL/ws/collab/$DOC_ID?userId=$ALICE_ID"
echo ""
echo "🌐 Open two browser tabs to test real-time collaboration:"
echo "   Tab 1: http://localhost:3000/doc/$DOC_ID (as Alice)"
echo "   Tab 2: http://localhost:3000/doc/$DOC_ID (switch to Bob)"
echo ""
