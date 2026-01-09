#!/bin/bash

echo "🧪 Testing Incremental Indexing"
echo "================================"
echo ""

# Start the server in background
echo "Starting MCP server with auto-indexing..."
./code-rag-mcp > /tmp/mcp_server.log 2>&1 &
SERVER_PID=$!

echo "Server PID: $SERVER_PID"
echo "Logs: tail -f /tmp/mcp_server.log"
echo ""

# Wait a bit for server to start
sleep 3

# Monitor indexing progress
echo "Monitoring indexing progress (press Ctrl+C to stop)..."
echo ""

while true; do
    if [ -f .indexing_state.json ]; then
        clear
        echo "📊 Indexing Progress"
        echo "===================="
        echo ""
        cat .indexing_state.json | jq -r '
            "Status: \(.status)",
            "Progress: \(.indexed_files)/\(.total_files) files (\((.indexed_files / .total_files * 100) | floor)%)",
            "Chunks: \(.total_chunks)",
            "Failed: \(.failed_files | length)",
            "Last Update: \(.last_update)"
        '
        echo ""
        
        STATUS=$(cat .indexing_state.json | jq -r '.status')
        if [ "$STATUS" = "completed" ]; then
            echo "✅ Indexing completed!"
            break
        fi
    else
        echo "⏳ Waiting for indexing to start..."
    fi
    
    sleep 2
done

# Kill server
echo ""
echo "Stopping server..."
kill $SERVER_PID

echo ""
echo "✅ Test complete!"
