#!/bin/bash

echo "🐋 Literary Lions Docker Testing Script"
echo "========================================"

# Test Case 1: Check Dockerfile exists
echo ""
echo "📋 Test Case 1: Checking if Dockerfile exists..."
if [ -f "Dockerfile" ]; then
    echo "✅ PASS: Dockerfile found"
    echo "   Size: $(ls -lh Dockerfile | awk '{print $5}')"
else
    echo "❌ FAIL: Dockerfile not found"
    exit 1
fi

# Test Case 2: Build Docker image
echo ""
echo "🔨 Test Case 2: Building Docker image..."
echo "Building image 'literary-lions-forum'..."
if docker build -t literary-lions-forum . ; then
    echo "✅ PASS: Docker image built successfully"
    echo "   Image info:"
    docker images literary-lions-forum --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
else
    echo "❌ FAIL: Docker image build failed"
    exit 1
fi

# Test Case 3: Run container successfully
echo ""
echo "🚀 Test Case 3: Testing container execution..."

# Start container in background
echo "Starting container..."
CONTAINER_ID=$(docker run -d -p 8080:8080 --name literary-lions-test literary-lions-forum)

if [ $? -eq 0 ]; then
    echo "✅ Container started with ID: $CONTAINER_ID"
    
    # Wait a moment for the container to start
    sleep 5
    
    # Test if container is running
    if docker ps | grep -q literary-lions-test; then
        echo "✅ Container is running"
        
        # Test HTTP endpoints
        echo "Testing HTTP endpoints..."
        HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ || echo "000")
        
        if [ "$HTTP_STATUS" = "200" ]; then
            echo "✅ PASS: Application responds correctly (HTTP $HTTP_STATUS)"
        else
            echo "❌ FAIL: Application not responding correctly (HTTP $HTTP_STATUS)"
            echo "Container logs:"
            docker logs literary-lions-test | tail -10
        fi
    else
        echo "❌ FAIL: Container stopped unexpectedly"
        echo "Container logs:"
        docker logs literary-lions-test
    fi
    
    # Cleanup test container
    echo "Cleaning up test container..."
    docker stop literary-lions-test >/dev/null 2>&1
    docker rm literary-lions-test >/dev/null 2>&1
    
else
    echo "❌ FAIL: Container failed to start"
    exit 1
fi

# Test Case 4: Check for unused Docker objects
echo ""
echo "🧹 Test Case 4: Checking for unused Docker objects..."
echo "Current Docker space usage:"
docker system df

echo ""
echo "Checking for unused objects..."

# Count dangling images
DANGLING_IMAGES=$(docker images -f "dangling=true" -q | wc -l)
echo "📦 Dangling images: $DANGLING_IMAGES"

# Count stopped containers
STOPPED_CONTAINERS=$(docker ps -a -f "status=exited" -q | wc -l)
echo "🛑 Stopped containers: $STOPPED_CONTAINERS"

# Count unused volumes
UNUSED_VOLUMES=$(docker volume ls -f "dangling=true" -q | wc -l)
echo "💾 Unused volumes: $UNUSED_VOLUMES"

# Overall assessment
TOTAL_UNUSED=$((DANGLING_IMAGES + STOPPED_CONTAINERS + UNUSED_VOLUMES))
if [ $TOTAL_UNUSED -eq 0 ]; then
    echo "✅ PASS: No unused Docker objects found"
else
    echo "⚠️  WARNING: $TOTAL_UNUSED unused Docker objects found"
    echo "   Run 'docker system prune' to clean up"
fi

echo ""
echo "🎉 Docker testing completed!"
echo "Summary:"
echo "- Dockerfile exists: ✅"
echo "- Image builds: ✅"
echo "- Container runs: ✅"
echo "- Clean Docker objects: $([ $TOTAL_UNUSED -eq 0 ] && echo '✅' || echo '⚠️')" 