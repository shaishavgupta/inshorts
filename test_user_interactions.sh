#!/bin/bash

BASE_URL="http://localhost:8080"

echo "=== Generating User Interactions for Trending API Testing ==="
echo ""

# ============================================
# Location 1: Mumbai, India (19.0760, 72.8777)
# ============================================

# User 1 - Multiple views on Article 1 (Mumbai)
echo "Recording interactions for User 1 in Mumbai..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_001",
    "article_id": "19aaddc0-7508-4659-9c32-2216107f8604",
    "event_type": "view",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# User 2 - Click on Article 1 (Mumbai)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_002",
    "article_id": "19aaddc0-7508-4659-9c32-2216107f8604",
    "event_type": "click",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# User 3 - View on Article 1 (Mumbai)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_003",
    "article_id": "19aaddc0-7508-4659-9c32-2216107f8604",
    "event_type": "view",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# User 4 - Click on Article 1 (Mumbai)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_004",
    "article_id": "19aaddc0-7508-4659-9c32-2216107f8604",
    "event_type": "click",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# ============================================
# Location 2: Delhi, India (28.6139, 77.2090)
# ============================================

# User 5 - View on Article 2 (Delhi)
echo "Recording interactions for User 5 in Delhi..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_005",
    "article_id": "099503a1-d4b6-460e-ad9c-19212d9dd9ac",
    "event_type": "view",
    "location": {
      "latitude": 28.6139,
      "longitude": 77.2090
    }
  }'

echo -e "\n"

# User 6 - Click on Article 2 (Delhi)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_006",
    "article_id": "099503a1-d4b6-460e-ad9c-19212d9dd9ac",
    "event_type": "click",
    "location": {
      "latitude": 28.6139,
      "longitude": 77.2090
    }
  }'

echo -e "\n"

# User 7 - View on Article 2 (Delhi)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_007",
    "article_id": "099503a1-d4b6-460e-ad9c-19212d9dd9ac",
    "event_type": "view",
    "location": {
      "latitude": 28.6139,
      "longitude": 77.2090
    }
  }'

echo -e "\n"

# ============================================
# Location 3: Bangalore, India (12.9716, 77.5946)
# ============================================

# User 8 - View on Article 3 (Bangalore)
echo "Recording interactions for User 8 in Bangalore..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_008",
    "article_id": "b90bc69d-3601-48f1-b897-fef0626e39dd",
    "event_type": "view",
    "location": {
      "latitude": 12.9716,
      "longitude": 77.5946
    }
  }'

echo -e "\n"

# User 9 - Click on Article 3 (Bangalore)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_009",
    "article_id": "b90bc69d-3601-48f1-b897-fef0626e39dd",
    "event_type": "click",
    "location": {
      "latitude": 12.9716,
      "longitude": 77.5946
    }
  }'

echo -e "\n"

# User 10 - View on Article 3 (Bangalore)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_010",
    "article_id": "b90bc69d-3601-48f1-b897-fef0626e39dd",
    "event_type": "view",
    "location": {
      "latitude": 12.9716,
      "longitude": 77.5946
    }
  }'

echo -e "\n"

# ============================================
# High Engagement Article (Multiple users, same article)
# ============================================

# Multiple users viewing/clicking Article 4 to make it trending
echo "Recording high engagement for Article 4..."
for i in {11..20}; do
  curl -X POST "${BASE_URL}/api/v1/interactions/record" \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"user_$(printf %03d $i)\",
      \"article_id\": \"7dfdd5c6-02c1-4247-baf7-122a1d6f1b46\",
      \"event_type\": \"$([ $((i % 2)) -eq 0 ] && echo 'click' || echo 'view')\",
      \"location\": {
        \"latitude\": $((19 + i % 5)).$((1000 + i * 100)),
        \"longitude\": $((72 + i % 5)).$((8000 + i * 100))
      }
    }"
  echo -e "\n"
done

# ============================================
# Mixed Location Testing (Same article, different locations)
# ============================================

# Article 5 viewed from multiple locations
echo "Recording Article 5 interactions from multiple locations..."

# Mumbai
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_021",
    "article_id": "53ae5cbc-d798-485d-8990-2d97070635b5",
    "event_type": "view",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# Delhi
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_022",
    "article_id": "53ae5cbc-d798-485d-8990-2d97070635b5",
    "event_type": "click",
    "location": {
      "latitude": 28.6139,
      "longitude": 77.2090
    }
  }'

echo -e "\n"

# Bangalore
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_023",
    "article_id": "53ae5cbc-d798-485d-8990-2d97070635b5",
    "event_type": "view",
    "location": {
      "latitude": 12.9716,
      "longitude": 77.5946
    }
  }'

echo -e "\n"

# Chennai (13.0827, 80.2707)
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_024",
    "article_id": "53ae5cbc-d798-485d-8990-2d97070635b5",
    "event_type": "click",
    "location": {
      "latitude": 13.0827,
      "longitude": 80.2707
    }
  }'

echo -e "\n"

# ============================================
# Additional Article Interactions
# ============================================

# Article 6: Palestinian director (4da35990-3118-47e5-940d-9ddcd771764a)
echo "Recording interactions for Palestinian director article..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_025",
    "article_id": "4da35990-3118-47e5-940d-9ddcd771764a",
    "event_type": "view",
    "location": {
      "latitude": 28.6139,
      "longitude": 77.2090
    }
  }'

echo -e "\n"

curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_026",
    "article_id": "4da35990-3118-47e5-940d-9ddcd771764a",
    "event_type": "click",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# Article 7: IIT Kanpur Techkriti (204f91d7-8dfe-4816-a6af-6ed9ebc53117)
echo "Recording interactions for IIT Kanpur Techkriti article..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_027",
    "article_id": "204f91d7-8dfe-4816-a6af-6ed9ebc53117",
    "event_type": "view",
    "location": {
      "latitude": 12.9716,
      "longitude": 77.5946
    }
  }'

echo -e "\n"

curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_028",
    "article_id": "204f91d7-8dfe-4816-a6af-6ed9ebc53117",
    "event_type": "click",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# Article 8: Hamas protest (c3f7765c-01c3-4147-8b01-7f983111ae3d) - High engagement
echo "Recording high engagement for Hamas protest article..."
for i in {29..35}; do
  curl -X POST "${BASE_URL}/api/v1/interactions/record" \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"user_$(printf %03d $i)\",
      \"article_id\": \"c3f7765c-01c3-4147-8b01-7f983111ae3d\",
      \"event_type\": \"$([ $((i % 2)) -eq 0 ] && echo 'click' || echo 'view')\",
      \"location\": {
        \"latitude\": $((19 + i % 3)).$((1000 + i * 50)),
        \"longitude\": $((72 + i % 3)).$((8000 + i * 50))
      }
    }"
  echo -e "\n"
done

# Article 9: US Senate Bhattacharya (59a772e8-9463-4078-933b-40d940bad8c0)
echo "Recording interactions for US Senate Bhattacharya article..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_036",
    "article_id": "59a772e8-9463-4078-933b-40d940bad8c0",
    "event_type": "view",
    "location": {
      "latitude": 28.6139,
      "longitude": 77.2090
    }
  }'

echo -e "\n"

curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_037",
    "article_id": "59a772e8-9463-4078-933b-40d940bad8c0",
    "event_type": "click",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# Article 10: Suryakumar Yadav flats (c1f79956-4b7c-4486-a0b1-ff24503e151d) - Mumbai related
echo "Recording interactions for Suryakumar Yadav flats article (Mumbai)..."
curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_038",
    "article_id": "c1f79956-4b7c-4486-a0b1-ff24503e151d",
    "event_type": "view",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_039",
    "article_id": "c1f79956-4b7c-4486-a0b1-ff24503e151d",
    "event_type": "click",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

curl -X POST "${BASE_URL}/api/v1/interactions/record" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_040",
    "article_id": "c1f79956-4b7c-4486-a0b1-ff24503e151d",
    "event_type": "view",
    "location": {
      "latitude": 19.0760,
      "longitude": 72.8777
    }
  }'

echo -e "\n"

# ============================================
# Test Trending API After Interactions
# ============================================

echo "=== Testing Trending API ==="
echo ""

# Test 1: Get trending without location
echo "1. Get trending news (no location filter):"
curl -X GET "${BASE_URL}/api/v1/news/trending?limit=10"

echo -e "\n\n"

# Test 2: Get trending for Mumbai location
echo "2. Get trending news for Mumbai (19.0760, 72.8777):"
curl -X GET "${BASE_URL}/api/v1/news/trending?lat=19.0760&lon=72.8777&limit=10"

echo -e "\n\n"

# Test 3: Get trending for Delhi location
echo "3. Get trending news for Delhi (28.6139, 77.2090):"
curl -X GET "${BASE_URL}/api/v1/news/trending?lat=28.6139&lon=77.2090&limit=10"

echo -e "\n\n"

# Test 4: Get trending for Bangalore location
echo "4. Get trending news for Bangalore (12.9716, 77.5946):"
curl -X GET "${BASE_URL}/api/v1/news/trending?lat=12.9716&lon=77.5946&limit=10"

echo -e "\n\n"

# Test 5: Get trending with custom limit
echo "5. Get trending news with limit=5:"
curl -X GET "${BASE_URL}/api/v1/news/trending?limit=5"

echo -e "\n"

echo "=== Done ==="

