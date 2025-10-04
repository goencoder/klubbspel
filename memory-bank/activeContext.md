# Active Context: Klubbspel v1.2.0 (In Development)

## Project Status
**Current Version**: v1.2.0 (Multi-sport support with 8 sports released)
**Next Release**: v1.3.0 - Ladder series format with architectural improvements
**Active Branch**: `codex/implement-ladder-series-format` (PR #22 - Ready to merge and tag as v1.3.0)

**Production Status**: Stable system serving Swedish clubs with comprehensive tournament management for table tennis and tennis.

**Open Source**: Released under open source license, welcoming community contributions.

## Recent Developments (October 2025)

### PR #22 - Ladder Series Format (October 4, 2025 - Ready for v1.3.0)
**Major Feature**: Tournament series now support both ELO and ladder-based ranking systems

**Implementation Complete**:
✅ Backend ladder standings calculation with two variants (Classic, Aggressive)
✅ Ladder penalty system for losing to lower-ranked players (Aggressive mode)
✅ Series format selection (ELO/Ladder) with variant configuration
✅ Complete UI implementation with ladder-specific leaderboards
✅ Comprehensive Playwright UI tests verifying penalty mechanics
✅ Session token expiration handling with proper error recovery
✅ All PR review comments addressed (14 initial + 5 architectural fixes)

**Architectural Improvements**:
✅ Named struct refactoring: `playerMatchStats` replaces 7+ anonymous structs
✅ Circular dependency eliminated: New `sessionManager.ts` module as mediator
✅ Error handling improved: Cursor.Close errors properly handled
✅ Warnings field added to ReportMatchResponse for graceful degradation
✅ Code quality: 0 lint issues, all tests passing

**Ladder Variants Implemented**:
- **Classic**: No penalty - winners climb, losers maintain position
- **Aggressive**: Penalty system - losers drop position when beaten by lower-ranked players

**Testing Validation** (Comprehensive UI smoke testing completed):
- Tested aggressive ladder penalty: Player losing to lower-ranked opponent drops position ✅
- Tested downward challenge: Lower-ranked player beating higher-ranked swaps positions ✅
- Verified match history: 3 matches recorded correctly ✅
- Confirmed leaderboard stats: Win rates calculated accurately (67% vs 33%) ✅

**Commits**:
- e9e109b: "fix: Address PR review comments - refactor structs and fix circular dependency"
- febbf98: "feat: Add session token expiration handling"
- 1e6e8e9: "fix: Address PR review comments - improve code quality and error handling"

**Status**: All code committed, pushed to origin, ready to merge and tag as v1.3.0

### v1.1.0 - Tennis Support (Completed)
- Added tennis as second supported sport
- Fixed 3 critical bugs (501 error, icon imports, translation namespace)
- Created CODEX_ADD_SPORTS.md comprehensive guide
- Sport-specific icons: CircleDot (table tennis), Circle (tennis)

### Previous Work - Scoring Improvements (fix/sport-scoring-improvements branch - Merged)
**Problem Identified**: User feedback revealed scoring limitations:
- Only table tennis showed "Sets to Play" field (hidden for other sports)
- Validation hints always showed "Best of 5" regardless of actual configuration
- No "Best of 7" option (tennis/squash sometimes need this)
- All sports forced to use identical validation rules

**Fixes Implemented**:
✅ Extended protobuf validation: `sets_to_play` now allows 3-7 (was 3-5)
✅ "Sets to Play" field now visible for ALL racket/paddle sports
✅ Added "Best of 7" option to dropdown menu
✅ Validation hints now dynamic: "Best of {setsToPlay}" instead of hardcoded
✅ TypeScript types updated with all new sports
✅ Translations added for bestOf7 (Swedish + English)

### v1.2.0 - Multi-Sport Support (Released - Merged in PR #21)
**Sports Added**: Badminton, Squash, Padel, Pickleball, Racquetball, Beach Tennis
**Status**: Released in v1.2.0
**Icons**: Wind (badminton), Zap (squash), Swords (padel), CircleDot (pickleball)
**Total Sports Supported**: 8 racket/paddle sports now available
**Date**: Merged and tagged before ladder implementation

## Critical Technical Debt Identified

### ⚠️ Sport-Specific Validation Framework Needed (High Priority)
**Current Problem**: All sports share `validateTableTennisScore()` function
- No sport-specific rules implemented
- All sports use identical validation: best-of-N sets, no draws allowed
- Squash cannot support draws (some leagues require this)
- No configuration for sport-specific scoring systems

**Impact on Future Sports**:
Cannot properly support these sports without refactoring:
- **Dart**: 301/501 checkout rules, double-in/double-out, leg scoring
- **Chess**: Draw by agreement, stalemate, threefold repetition, time controls
- **Fishing**: Weight-based scoring, catch-and-release rules
- **Golf**: Stroke play vs match play, handicaps, par validation
- **Tennis (advanced)**: Tiebreaker rules, advantage sets, no-ad scoring
- **Squash (proper)**: Draw support, different point systems

**Recommended Solution**: 
1. Create `SportValidator` interface in Go
2. Implement sport-specific validators (TableTennisValidator, ChessValidator, etc.)
3. Add `SportConfig` message to protobuf with allows_draws, scoring_system, etc.
4. Update series creation to allow sport-specific configuration

## Core Capabilities Delivered
- **Complete Tournament Management**: End-to-end series creation, player registration, match reporting, and live leaderboards
- **Robust Error Handling**: Comprehensive error management with proper request cancellation handling and user-friendly error messages
- **Production Quality**: Full test coverage (unit, integration, and UI tests), internationalization support, and secure authentication
- **Developer Experience**: Well-documented build system, clear architecture patterns, and comprehensive development guidelines
- **Data Integrity**: Player merging system, duplicate detection, and validated match scoring rules

## Active Development Focus

### Multi-Sport Framework
**Status**: Fully implemented with 8 racket/paddle sports supported (Table Tennis, Tennis, Badminton, Squash, Padel, Pickleball, Racquetball, Beach Tennis).
**Extension Points**: Ready for community contributions to add more sports or implement sport-specific validation rules.
**Design**: Extensible service layer with Sport enum and flexible scoring profiles (sets, scoreline, stroke card, weigh-in).

### Mobile Experience
**Status**: Fully responsive web application optimized for mobile browsers.
**Future Considerations**: Progressive Web App (PWA) capabilities or native mobile apps based on community feedback and usage patterns.

### Advanced Features
**Status**: Core tournament functionality complete with dual ranking systems (ELO and Ladder with Classic/Aggressive variants), multi-sport support (8 sports), and comprehensive series management.
**Extension Opportunities**: Bracket tournaments, advanced scheduling, elimination formats, and sport-specific validation rules can be added as community needs arise.

## Architecture Principles

### Development Quality Standards
- **Never Cancel Long Operations**: Build commands have specific timing expectations (documented in copilot-instructions.md)
- **Comprehensive Testing**: Three-tier testing strategy (unit, integration, UI) ensures reliability
- **Type Safety First**: Protocol buffers provide end-to-end type safety from database to UI
- **Localization Required**: Full Swedish/English internationalization across all user-facing text

### Code Organization
- **Clean Architecture**: Clear separation between service, repository, and transport layers
- **API-First Design**: Protocol buffer definitions serve as the single source of truth for all interfaces
- **Error Handling**: Structured error responses with localization keys for user-friendly messages
- **Validation Layers**: Dual validation (client and server) ensures data integrity

### User Experience Philosophy
- **Swedish-First**: Native Swedish interface designed for Swedish table tennis clubs
- **Immediate Feedback**: Real-time leaderboard updates and instant match result processing
- **Data Integrity**: Comprehensive validation prevents invalid data entry
- **Mobile-Responsive**: Touch-optimized interface for all screen sizes

## Project Architecture Highlights

### Technical Decisions
- **Protocol Buffers**: Provides type safety, versioning, and automatic code generation
- **gRPC + REST Gateway**: High-performance gRPC with REST compatibility for broad client support
- **MongoDB**: Document model fits flexible tournament structures with built-in indexing
- **Repository Pattern**: Clean data access abstraction enables thorough testing
- **Host Development**: Fast iteration with MongoDB in Docker, services on host machine

### Feature Highlights
- **Player Duplicate Management**: Intelligent normalization and merging prevents data fragmentation
- **Real-time Leaderboards**: Automatic ELO calculation with instant ranking updates
- **Table Tennis Validation**: Sport-specific scoring rules ensure match result accuracy
- **CSV Export**: Administrative data export for league reporting and external analysis
- **Club Administration**: Role-based access control with admin and member permissions

### Production Readiness
- **Automated Testing**: Comprehensive test suite with unit, integration, and UI coverage
- **Security**: JWT authentication, input validation, and parameterized database queries
- **Monitoring**: Health checks, structured logging, and error tracking
- **Deployment**: Docker-based deployment with infrastructure-as-code configuration
- **Documentation**: Extensive developer guides and architectural documentation