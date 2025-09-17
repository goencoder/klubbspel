import styled from 'styled-components';
import { colors, spacing } from '../Styles';

// =============================================================================
// PAGE-LEVEL STYLED COMPONENTS
// =============================================================================

// Re-export shared empty state for page usage
export { SharedEmptyState, HoverCard } from '../components/Styles';

// =============================================================================
// PAGE-LEVEL STYLED COMPONENTS
// =============================================================================

// Page wrapper - consistent spacing and layout for all pages
export const PageWrapper = styled.div`
  /* Preserves Tailwind 'space-y-6' pattern */
  & > * + * {
    margin-top: ${spacing.xl};
  }
`;

// Page header section - title, subtitle, and primary action pattern
export const PageHeaderSection = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing.lg};
  
  @media (min-width: 640px) {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
`;

// Header content - title and subtitle grouping
export const HeaderContent = styled.div`
  h1 {
    font-size: 1.875rem;
    font-weight: 700;
    color: ${colors.text.primary};
    margin: 0 0 ${spacing.sm} 0;
  }
  
  p {
    color: ${colors.text.secondary};
    margin: 0;
  }
`;

// Search section - consistent search input styling across pages
export const SearchSection = styled.div`
  position: relative;
  max-width: 28rem;
  
  svg {
    position: absolute;
    left: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    color: ${colors.text.secondary};
    pointer-events: none;
  }
  
  input {
    padding-left: 2.5rem;
  }
`;

// Content grid - responsive grid layout for cards/items with consistent spacing
export const ContentGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr;
  gap: ${spacing.lg}; /* 16px - consistent across all pages */
  
  @media (min-width: 768px) {
    grid-template-columns: repeat(2, 1fr);
  }
  
  @media (min-width: 1024px) {
    grid-template-columns: repeat(3, 1fr);
  }
`;

// Loading grid - skeleton loading state with same grid layout
export const LoadingGrid = styled(ContentGrid)`
  /* Inherits responsive grid layout with consistent spacing */
`;

// Action button group - consistent spacing for action buttons
export const ActionGroup = styled.div`
  display: flex;
  align-items: center;
  gap: ${spacing.xs};
`;