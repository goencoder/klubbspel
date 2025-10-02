#!/bin/bash

set -e

# Get absolute path of script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Populating Development Environment with Test Data${NC}"

# Check if dev environment is running
if ! curl -s http://localhost:8080/v1/clubs >/dev/null 2>&1; then
    echo -e "${YELLOW}Development environment not running. Starting it now...${NC}"
    cd "$PROJECT_ROOT"
    make dev-start
    
    # Wait for server to be ready
    echo -e "${YELLOW}Waiting for server to start...${NC}"
    for i in {1..30}; do
        if curl -s http://localhost:8080/v1/clubs >/dev/null 2>&1; then
            echo -e "${GREEN}Server is ready!${NC}"
            break
        fi
        if [ $i -eq 30 ]; then
            echo -e "${RED}Server failed to start within 30 seconds${NC}"
            exit 1
        fi
        sleep 1
    done
fi

echo -e "${YELLOW}Creating test data via API calls...${NC}"

# Function to make API calls with better error handling
make_api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "  Creating: ${description}"
    
    if [ "$method" = "POST" ]; then
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$data" \
            "http://localhost:8080$endpoint")
    else
        response=$(curl -s "http://localhost:8080$endpoint")
    fi
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo -e "    ${GREEN}✓${NC} API call completed"
        echo "$response"
        return 0
    fi
    
    # Check if response is valid JSON
    if echo "$response" | jq . >/dev/null 2>&1; then
        echo -e "    ${GREEN}✓${NC} Success"
        echo "$response"
    else
        echo -e "    ${RED}✗${NC} Failed or non-JSON response"
        echo "    Response: $response"
        return 1
    fi
}

# Store created IDs
declare -a CLUB_IDS
declare -a PLAYER_IDS
declare -a SERIES_IDS

# Create clubs
echo -e "${YELLOW}📋 Creating Clubs...${NC}"
clubs=('Stockholm TK' 'Malmö Bordtennis' 'Göteborg Pingis')

for club_name in "${clubs[@]}"; do
    club_data="{\"name\": \"$club_name\"}"
    response=$(make_api_call "POST" "/v1/clubs" "$club_data" "$club_name")
    
    # Extract club ID - try with jq first, fallback to basic parsing
    if command -v jq &> /dev/null && echo "$response" | jq . >/dev/null 2>&1; then
        club_id=$(echo "$response" | jq -r '.club.id')
    else
        # Basic parsing fallback - look for id field in JSON
        club_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    
    CLUB_IDS+=("$club_id")
    echo -e "    ${BLUE}Club ID: $club_id${NC}"
done

echo ""

# Create players
echo -e "${YELLOW}👥 Creating Players...${NC}"
declare -a player_data=(
    "Alice Johnson:0"
    "Bob Smith:0"
    "Charlie Brown:0"
    "Diana Prince:1"
    "Erik Nilsson:1"
    "Fiona Andersson:2"
)

for player_entry in "${player_data[@]}"; do
    IFS=':' read -r player_name club_index <<< "$player_entry"
    club_id="${CLUB_IDS[$club_index]}"
    
    player_json="{\"displayName\": \"$player_name\", \"clubId\": \"$club_id\"}"
    response=$(make_api_call "POST" "/v1/players" "$player_json" "$player_name")
    
    # Extract player ID
    if command -v jq &> /dev/null && echo "$response" | jq . >/dev/null 2>&1; then
        player_id=$(echo "$response" | jq -r '.player.id')
    else
        player_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    
    PLAYER_IDS+=("$player_id")
    echo -e "    ${BLUE}Player ID: $player_id${NC}"
done

echo ""

# Create series
echo -e "${YELLOW}🏆 Creating Tournament Series...${NC}"
declare -a series_data=(
    "Spring Championship 2024:0:SERIES_VISIBILITY_CLUB_ONLY:30"
    "Summer Open 2024:1:SERIES_VISIBILITY_OPEN:45"
    "Autumn Cup 2024:2:SERIES_VISIBILITY_CLUB_ONLY:21"
)

for series_entry in "${series_data[@]}"; do
    IFS=':' read -r title club_index visibility days <<< "$series_entry"
    club_id="${CLUB_IDS[$club_index]}"
    
    # Calculate dates - using compatible date format for macOS
    starts_at=$(date -u -j -v+1d +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "tomorrow" +"%Y-%m-%dT%H:%M:%SZ")
    ends_at=$(date -u -j -v+${days}d +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+${days} days" +"%Y-%m-%dT%H:%M:%SZ")
    
    series_json="{
        \"clubId\": \"$club_id\",
        \"title\": \"$title\",
        \"startsAt\": \"$starts_at\",
        \"endsAt\": \"$ends_at\",
        \"visibility\": \"$visibility\"
    }"
    
    response=$(make_api_call "POST" "/v1/series" "$series_json" "$title")
    
    # Extract series ID
    if command -v jq &> /dev/null && echo "$response" | jq . >/dev/null 2>&1; then
        series_id=$(echo "$response" | jq -r '.series.id')
    else
        series_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    
    SERIES_IDS+=("$series_id")
    echo -e "    ${BLUE}Series ID: $series_id${NC}"
done

echo ""

# Create matches for the first series (Spring Championship)
echo -e "${YELLOW}🏓 Creating Sample Matches...${NC}"
spring_series_id="${SERIES_IDS[0]}"

declare -a match_data=(
    "0:1:3:2:-2:Alice beats Bob 3-2"
    "2:0:1:3:-1:Alice beats Charlie 3-1"
    "1:2:3:0:0:Bob beats Charlie 3-0"
)

for match_entry in "${match_data[@]}"; do
    IFS=':' read -r player_a_idx player_b_idx score_a score_b hours_offset description <<< "$match_entry"
    
    player_a_id="${PLAYER_IDS[$player_a_idx]}"
    player_b_id="${PLAYER_IDS[$player_b_idx]}"
    
    # Calculate match time - using compatible date format for macOS
    if [ "$hours_offset" -eq 0 ]; then
        played_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    else
        played_at=$(date -u -j -v${hours_offset}H +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "${hours_offset} hours" +"%Y-%m-%dT%H:%M:%SZ")
    fi
    
    match_json="{
        \"seriesId\": \"$spring_series_id\",
        \"playerAId\": \"$player_a_id\",
        \"playerBId\": \"$player_b_id\",
        \"scoreA\": $score_a,
        \"scoreB\": $score_b,
        \"playedAt\": \"$played_at\"
    }"
    
    response=$(make_api_call "POST" "/v1/matches:report" "$match_json" "$description")
    
    # Extract match ID
    if command -v jq &> /dev/null && echo "$response" | jq . >/dev/null 2>&1; then
        match_id=$(echo "$response" | jq -r '.matchId')
    else
        match_id=$(echo "$response" | grep -o '"matchId":"[^"]*"' | cut -d'"' -f4)
    fi
    
    echo -e "    ${BLUE}Match ID: $match_id${NC}"
done

echo ""
echo -e "${GREEN}🎉 Test data population complete!${NC}"
echo ""
echo -e "${BLUE}📊 Summary:${NC}"
echo -e "  Clubs: ${#CLUB_IDS[@]}"
echo -e "  Players: ${#PLAYER_IDS[@]}"
echo -e "  Series: ${#SERIES_IDS[@]}"
echo -e "  Matches: 3"
echo ""
echo -e "${BLUE}🌐 Available endpoints:${NC}"
echo -e "  API: http://localhost:8080"
echo -e "  OpenAPI/Swagger: http://localhost:8081"
echo ""
echo -e "${BLUE}🔍 Try these API calls:${NC}"
echo -e "  curl http://localhost:8080/v1/clubs"
echo -e "  curl http://localhost:8080/v1/players"
echo -e "  curl http://localhost:8080/v1/series"
echo -e "  curl http://localhost:8080/v1/series/${SERIES_IDS[0]}/leaderboard"
echo -e "  curl http://localhost:8080/v1/series/${SERIES_IDS[0]}/matches"
echo ""
echo -e "${YELLOW}💡 Your development environment is ready for UI testing!${NC}"
