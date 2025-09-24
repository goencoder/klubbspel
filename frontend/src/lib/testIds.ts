/**
 * Deterministic Test IDs for Klubbspel UI Components
 * 
 * This file contains all test IDs used throughout the application.
 * MUST BE USED: All interactive UI components must have deterministic IDs
 * for reliable testing and debugging.
 * 
 * Conventions:
 * - Use kebab-case for all IDs
 * - Include context in the ID (e.g., 'match-', 'player-', 'series-')
 * - Use functions for indexed items (rows, options, etc.)
 * - Keep IDs descriptive and self-documenting
 */

export const testIds = {
  // ===========================================
  // MATCH REPORTING - Core workflow
  // ===========================================
  matchReport: {
    dialog: 'match-report-dialog',
    title: 'match-report-title',
    form: 'match-report-form',
    playerASelector: 'match-player-a-selector',
    playerBSelector: 'match-player-b-selector',
    playerALabel: 'match-player-a-label',
    playerBLabel: 'match-player-b-label',
    scoreA: 'match-score-a',
    scoreB: 'match-score-b',
    scoreALabel: 'match-score-a-label',
    scoreBLabel: 'match-score-b-label',
    date: 'match-date',
    time: 'match-time',
    dateLabel: 'match-date-label',
    timeLabel: 'match-time-label',
    keepOpenSwitch: 'match-keep-open-switch',
    keepOpenLabel: 'match-keep-open-label',
    validationHints: 'match-validation-hints',
    submitBtn: 'match-submit-btn',
    cancelBtn: 'match-cancel-btn',
    closeBtn: 'match-close-btn'
  },

  // ===========================================
  // PLAYER SELECTION - Dropdowns and lists
  // ===========================================
  playerSelector: {
    trigger: (context: string) => `player-selector-trigger-${context}`,
    popover: (context: string) => `player-selector-popover-${context}`,
    searchInput: (context: string) => `player-selector-search-${context}`,
    optionsList: (context: string) => `player-selector-options-${context}`,
    option: (index: number) => `player-option-${index}`,
    optionByName: (name: string) => `player-option-${name.toLowerCase().replace(/\s+/g, '-')}`,
    noResults: (context: string) => `player-selector-no-results-${context}`,
    loading: (context: string) => `player-selector-loading-${context}`
  },

  // ===========================================
  // MATCHES LIST - Match results table
  // ===========================================
  matchesList: {
    container: 'matches-list-container',
    header: 'matches-list-header',
    title: 'matches-list-title',
    exportBtn: 'matches-export-btn',
    table: 'matches-table',
    tableHeader: 'matches-table-header',
    tableBody: 'matches-table-body',
    row: (index: number) => `match-row-${index}`,
    playerACell: (index: number) => `match-player-a-${index}`,
    playerBCell: (index: number) => `match-player-b-${index}`,
    scoreCell: (index: number) => `match-score-${index}`,
    dateCell: (index: number) => `match-date-${index}`,
    actionsCell: (index: number) => `match-actions-${index}`,
    editBtn: (index: number) => `match-edit-btn-${index}`,
    deleteBtn: (index: number) => `match-delete-btn-${index}`,
    winnerIcon: (index: number) => `match-winner-icon-${index}`,
    emptyState: 'matches-empty-state',
    loadMoreBtn: 'matches-load-more-btn'
  },

  // ===========================================
  // LEADERBOARD - Rankings table
  // ===========================================
  leaderboard: {
    container: 'leaderboard-container',
    header: 'leaderboard-header',
    title: 'leaderboard-title',
    description: 'leaderboard-description',
    exportBtn: 'leaderboard-export-btn',
    table: 'leaderboard-table',
    tableHeader: 'leaderboard-table-header',
    tableBody: 'leaderboard-table-body',
    row: (index: number) => `leaderboard-row-${index}`,
    rankCell: (index: number) => `leaderboard-rank-${index}`,
    playerCell: (index: number) => `leaderboard-player-${index}`,
    ratingCell: (index: number) => `leaderboard-rating-${index}`,
    gamesCell: (index: number) => `leaderboard-games-${index}`,
    winsCell: (index: number) => `leaderboard-wins-${index}`,
    lossesCell: (index: number) => `leaderboard-losses-${index}`,
    winrateCell: (index: number) => `leaderboard-winrate-${index}`,
    rankIcon: (index: number) => `leaderboard-rank-icon-${index}`,
    emptyState: 'leaderboard-empty-state'
  },

  // ===========================================
  // SERIES DETAIL PAGE - Tournament management
  // ===========================================
  seriesDetail: {
    container: 'series-detail-container',
    backBtn: 'series-back-btn',
    header: 'series-detail-header',
    infoCard: 'series-info-card',
    title: 'series-detail-title',
    description: 'series-detail-description',
    visibilityBadge: 'series-visibility-badge',
    reportMatchBtn: 'series-report-match-btn',
    leaderboardBtn: 'series-leaderboard-btn',
    infoGrid: 'series-info-grid',
    durationInfo: 'series-duration-info',
    sportInfo: 'series-sport-info',
    formatInfo: 'series-format-info',
    clubInfo: 'series-club-info',
    contentTabs: 'series-content-tabs',
    matchesTab: 'series-matches-tab',
    leaderboardTab: 'series-leaderboard-tab'
  },

  // ===========================================
  // SERIES CREATION - New tournament form
  // ===========================================
  seriesCreation: {
    dialog: 'create-series-dialog',
    title: 'create-series-title',
    description: 'create-series-description',
    form: 'create-series-form',
    nameInput: 'series-name-input',
    nameLabel: 'series-name-label',
    startDateInput: 'series-start-date-input',
    startDateLabel: 'series-start-date-label',
    endDateInput: 'series-end-date-input',
    endDateLabel: 'series-end-date-label',
    sportSelect: 'series-sport-select',
    sportLabel: 'series-sport-label',
    formatSelect: 'series-format-select',
    formatLabel: 'series-format-label',
    visibilitySelect: 'series-visibility-select',
    visibilityLabel: 'series-visibility-label',
    submitBtn: 'create-series-submit-btn',
    cancelBtn: 'create-series-cancel-btn'
  },

  // ===========================================
  // SERIES LIST PAGE - Tournament listing
  // ===========================================
  seriesList: {
    container: 'series-list-container',
    header: 'series-list-header',
    title: 'series-list-title',
    subtitle: 'series-list-subtitle',
    createBtn: 'create-series-btn',
    searchInput: 'series-search-input',
    clubFilter: 'series-club-filter',
    filterLabel: 'series-filter-label',
    grid: 'series-grid',
    card: (index: number) => `series-card-${index}`,
    cardTitle: (index: number) => `series-card-title-${index}`,
    cardDates: (index: number) => `series-card-dates-${index}`,
    cardBadges: (index: number) => `series-card-badges-${index}`,
    viewBtn: (index: number) => `series-view-btn-${index}`,
    emptyState: 'series-empty-state'
  },

  // ===========================================
  // PLAYER CREATION - New player form
  // ===========================================
  playerCreation: {
    dialog: 'create-player-dialog',
    title: 'create-player-title',
    form: 'create-player-form',
    nameInput: 'player-name-input',
    nameLabel: 'player-name-label',
    clubSelect: 'player-club-select',
    clubLabel: 'player-club-label',
    submitBtn: 'create-player-submit-btn',
    cancelBtn: 'create-player-cancel-btn',
    confirmDialog: 'player-confirm-dialog',
    confirmTitle: 'player-confirm-title',
    similarPlayersList: 'similar-players-list',
    useSimilarBtn: 'use-similar-player-btn',
    createNewBtn: 'create-new-player-btn'
  },

  // ===========================================
  // CLUBS LIST PAGE - Club browsing
  // ===========================================
  clubsList: {
    container: 'clubs-list-container',
    header: 'clubs-list-header',
    title: 'clubs-list-title',
    subtitle: 'clubs-list-subtitle',
    createBtn: 'create-club-btn',
    searchInput: 'clubs-search-input',
    grid: 'clubs-grid',
    card: (index: number) => `club-card-${index}`,
    cardTitle: (index: number) => `club-card-title-${index}`,
    cardDescription: (index: number) => `club-card-description-${index}`,
    cardStats: (index: number) => `club-card-stats-${index}`,
    cardActions: (index: number) => `club-card-actions-${index}`,
    viewBtn: (index: number) => `club-view-btn-${index}`,
    joinBtn: (index: number) => `club-join-btn-${index}`,
    editBtn: (index: number) => `club-edit-btn-${index}`,
    deleteBtn: (index: number) => `club-delete-btn-${index}`,
    emptyState: 'clubs-empty-state'
  },

  // ===========================================
  // CLUB CREATION - New club form
  // ===========================================
  clubCreation: {
    dialog: 'create-club-dialog',
    title: 'create-club-title',
    form: 'create-club-form',
    nameInput: 'club-name-input',
    nameLabel: 'club-name-label',
    descriptionInput: 'club-description-input',
    descriptionLabel: 'club-description-label',
    submitBtn: 'create-club-submit-btn',
    cancelBtn: 'create-club-cancel-btn'
  },

  // ===========================================
  // CLUB DETAIL PAGE - Club management
  // ===========================================
  clubDetail: {
    container: 'club-detail-container',
    backBtn: 'club-back-btn',
    header: 'club-detail-header',
    title: 'club-detail-title',
    description: 'club-detail-description',
    joinBtn: 'club-join-btn',
    leaveBtn: 'club-leave-btn',
    signInBtn: 'club-signin-btn',
    tabs: 'club-detail-tabs',
    overviewTab: 'club-overview-tab',
    seriesTab: 'club-series-tab',
    membersTab: 'club-members-tab',
    mergeTab: 'club-merge-tab',
    statsGrid: 'club-stats-grid',
    activeSeriesCard: 'club-active-series-card',
    totalSeriesCard: 'club-total-series-card',
    clubTypeCard: 'club-type-card'
  },

  // ===========================================
  // LEADERBOARD PAGE - Main rankings
  // ===========================================
  leaderboardPage: {
    container: 'leaderboard-page-container',
    header: 'leaderboard-page-header',
    title: 'leaderboard-page-title',
    subtitle: 'leaderboard-page-subtitle',
    seriesSelectCard: 'series-select-card',
    seriesSelectTitle: 'series-select-title',
    seriesSelect: 'series-select',
    seriesSelectTrigger: 'series-select-trigger',
    seriesSelectContent: 'series-select-content',
    seriesOption: (seriesId: string) => `series-option-${seriesId}`,
    selectedSeriesCard: 'selected-series-card',
    selectedSeriesTitle: 'selected-series-title'
  },

  // ===========================================
  // NAVIGATION - Global navigation elements
  // ===========================================
  navigation: {
    header: 'main-header',
    logo: 'main-logo',
    navMenu: 'main-nav-menu',
    seriesLink: 'nav-series-link',
    clubsLink: 'nav-clubs-link',
    playersLink: 'nav-players-link',
    leaderboardLink: 'nav-leaderboard-link',
    clubFilter: 'nav-club-filter',
    userMenu: 'nav-user-menu',
    userAvatar: 'nav-user-avatar'
  },

  // ===========================================
  // COMMON ELEMENTS - Shared UI components
  // ===========================================
  common: {
    loadingSpinner: 'loading-spinner',
    errorMessage: 'error-message',
    successMessage: 'success-message',
    backButton: 'back-button',
    submitButton: 'submit-button',
    cancelButton: 'cancel-button',
    confirmButton: 'confirm-button',
    deleteButton: 'delete-button',
    editButton: 'edit-button',
    viewButton: 'view-button',
    exportButton: 'export-button',
    searchInput: 'search-input',
    filterSelect: 'filter-select',
    pagination: 'pagination',
    emptyState: 'empty-state'
  }
} as const

/**
 * Helper function to create data-testid attributes
 * Usage: <div {...dataTestId('match-report-dialog')}>
 */
export const dataTestId = (id: string) => ({
  'data-testid': id
})

/**
 * Helper function for indexed items with base ID
 * Usage: getIndexedId('match-row', 0) => 'match-row-0'
 */
export const getIndexedId = (baseId: string, index: number) => `${baseId}-${index}`

/**
 * Helper function to slugify names for consistent IDs
 * Usage: slugifyForId('Anna Andersson') => 'anna-andersson'
 */
export const slugifyForId = (text: string) => 
  text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

/**
 * Type-safe test ID getter with autocomplete
 */
export type TestId = keyof typeof testIds | string

export default testIds