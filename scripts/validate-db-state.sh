#!/bin/bash

# Database State Validation Script
# Helps detect stale data issues before they cause problems

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔍 Database State Validation${NC}"

# Function to check if MongoDB is running
check_mongo_running() {
    if docker ps | grep -q "pingis.*mongo"; then
        echo -e "${GREEN}✓${NC} MongoDB container is running"
        return 0
    else
        echo -e "${RED}✗${NC} MongoDB container is not running"
        return 1
    fi
}

# Function to get collection counts
get_collection_counts() {
    local db_name="$1"
    echo -e "${YELLOW}Collection counts in $db_name:${NC}"
    
    # Get counts for each collection
    docker exec $(docker ps --filter "name=mongo" --format "{{.ID}}" | head -1) \
        mongosh "$db_name" --quiet --eval "
        const collections = ['clubs', 'players', 'series', 'matches'];
        collections.forEach(col => {
            const count = db[col].countDocuments({});
            print(\`  \${col}: \${count}\`);
        });
    " 2>/dev/null || echo "  Could not connect to database"
}

# Function to check for test database pollution
check_test_pollution() {
    echo -e "${YELLOW}Checking for test data pollution...${NC}"
    
    # Check if test database exists
    if docker exec $(docker ps --filter "name=mongo" --format "{{.ID}}" | head -1) \
        mongosh --quiet --eval "db.adminCommand('listDatabases').databases.forEach(d => print(d.name))" 2>/dev/null | grep -q "pingis_test"; then
        echo -e "${RED}⚠️${NC} Test database 'pingis_test' exists in main MongoDB"
        get_collection_counts "pingis_test"
        echo -e "${YELLOW}Consider running: make test-clean${NC}"
        return 1
    else
        echo -e "${GREEN}✓${NC} No test database pollution detected"
        return 0
    fi
}

# Function to validate development data
validate_dev_data() {
    echo -e "${YELLOW}Development database state:${NC}"
    get_collection_counts "pingis"
    
    # Check for suspicious data patterns
    local club_count=$(docker exec $(docker ps --filter "name=mongo" --format "{{.ID}}" | head -1) \
        mongosh "pingis" --quiet --eval "db.clubs.countDocuments({})" 2>/dev/null || echo "0")
    
    if [ "$club_count" -gt 50 ]; then
        echo -e "${YELLOW}⚠️${NC} High club count ($club_count) - possible test data leakage"
        return 1
    fi
    
    return 0
}

# Main validation
main() {
    local exit_code=0
    
    if check_mongo_running; then
        if ! check_test_pollution; then
            exit_code=1
        fi
        
        if ! validate_dev_data; then
            exit_code=1
        fi
    else
        echo -e "${BLUE}ℹ️${NC} MongoDB not running - state validation skipped"
    fi
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ Database state validation passed${NC}"
    else
        echo -e "${RED}❌ Database state validation found issues${NC}"
        echo -e "${YELLOW}💡 Suggested actions:${NC}"
        echo -e "  • Run ${BLUE}make clean-all${NC} for complete cleanup"
        echo -e "  • Run ${BLUE}make test-clean${NC} to clean test data"
        echo -e "  • Run ${BLUE}make dev-clean${NC} to clean development data"
    fi
    
    return $exit_code
}

# Run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
