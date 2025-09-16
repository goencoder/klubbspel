import styled from 'styled-components';
import { colors, spacing, radii, shadows } from '../Styles';

// =============================================================================
// SHARED COMPONENT PATTERNS - Promoted from Leaf Folders
// =============================================================================

// Empty State - Standardized pattern for no-data scenarios
// Promotes similar patterns from MatchesList, PlayerSelector, and page components
export const SharedEmptyState = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: ${spacing['2xl']};
  text-align: center;
  
  /* Icon styling */
  svg {
    margin-bottom: ${spacing.lg};
    color: ${colors.text.secondary};
  }
  
  /* Title styling */
  h3 {
    font-size: 1.125rem;
    font-weight: 600;
    color: ${colors.text.primary};
    margin-bottom: ${spacing.sm};
  }
  
  /* Description styling */
  p {
    color: ${colors.text.secondary};
    margin: 0;
  }
`;

// Card component - Enhanced base card that complements UI library
// Promotes card patterns used across multiple components
export const StyledCard = styled.div`
  background: ${colors.ui.bgCard};
  border: 1px solid ${colors.ui.border};
  border-radius: ${radii.lg};
  box-shadow: ${shadows.sm};
  overflow: hidden;
  transition: box-shadow 0.2s ease;
  
  &:hover {
    box-shadow: ${shadows.md};
  }
  
  /* Card padding variants */
  &.padded {
    padding: ${spacing.xl};
  }
  
  &.compact {
    padding: ${spacing.lg};
  }
`;

// Form Section - Common form layout pattern
// Promotes form patterns found in dialog components
export const FormSection = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing.lg};
  
  /* Form row layout */
  .form-row {
    display: flex;
    flex-direction: column;
    gap: ${spacing.sm};
  }
  
  /* Form group with label and input */
  .form-group {
    display: flex;
    flex-direction: column;
    gap: ${spacing.xs};
    
    label {
      font-size: 0.875rem;
      font-weight: 500;
      color: ${colors.text.primary};
    }
    
    input, textarea, select {
      background: ${colors.ui.bg};
      border: 1px solid ${colors.ui.border};
      border-radius: ${radii.md};
      padding: ${spacing.sm} ${spacing.md};
      color: ${colors.text.primary};
      
      &:focus {
        outline: none;
        border-color: ${colors.ui.focus};
        box-shadow: 0 0 0 3px ${colors.ui.focus}20;
      }
      
      &::placeholder {
        color: ${colors.text.secondary};
      }
    }
  }
`;

// Data List - Pattern for lists with consistent item styling
// Promotes list patterns found across multiple components
export const DataList = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing.sm};
`;

// Data List Item - Individual list item with hover states
export const DataListItem = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: ${spacing.md};
  border: 1px solid ${colors.ui.border};
  border-radius: ${radii.md};
  background: ${colors.ui.bg};
  transition: background-color 0.2s ease;
  
  &:hover {
    background: ${colors.ui.bgSubtle};
  }
  
  /* Content area */
  .content {
    display: flex;
    align-items: center;
    gap: ${spacing.md};
    flex: 1;
  }
  
  /* Icon area */
  .icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border-radius: ${radii.md};
    background: ${colors.brand.primary}10;
    color: ${colors.brand.primary};
  }
  
  /* Text content */
  .text {
    display: flex;
    flex-direction: column;
    gap: ${spacing.xs};
    
    .title {
      font-weight: 500;
      color: ${colors.text.primary};
    }
    
    .subtitle {
      font-size: 0.875rem;
      color: ${colors.text.secondary};
    }
  }
  
  /* Actions area */
  .actions {
    display: flex;
    align-items: center;
    gap: ${spacing.xs};
  }
`;

// Loading Container - Consistent loading state layout
export const LoadingContainer = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  padding: ${spacing['2xl']};
  min-height: 200px;
`;

// Content Header - Shared header pattern for sections within components
export const ContentHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: ${spacing.lg};
  border-bottom: 1px solid ${colors.ui.border};
  margin-bottom: ${spacing.lg};
  
  h2 {
    font-size: 1.25rem;
    font-weight: 600;
    color: ${colors.text.primary};
    margin: 0;
  }
  
  .subtitle {
    font-size: 0.875rem;
    color: ${colors.text.secondary};
    margin-top: ${spacing.xs};
  }
`;

// Badge Container - For status indicators and tags
export const BadgeContainer = styled.div`
  display: inline-flex;
  align-items: center;
  gap: ${spacing.xs};
  padding: ${spacing.xs} ${spacing.sm};
  background: ${colors.ui.bgSubtle};
  border: 1px solid ${colors.ui.border};
  border-radius: ${radii.pill};
  font-size: 0.75rem;
  font-weight: 500;
  color: ${colors.text.secondary};
  
  /* Variant styles */
  &.success {
    background: ${colors.ui.success}20;
    color: ${colors.ui.success};
    border-color: ${colors.ui.success}40;
  }
  
  &.warning {
    background: ${colors.ui.warning}20;
    color: ${colors.ui.warning};
    border-color: ${colors.ui.warning}40;
  }
  
  &.danger {
    background: ${colors.ui.danger}20;
    color: ${colors.ui.danger};
    border-color: ${colors.ui.danger}40;
  }
`;

// Hover Card - Card component with hover effects
// Promotes the common "hover:shadow-md transition-shadow" pattern
export const HoverCard = styled(StyledCard)`
  transition: box-shadow 0.2s ease-in-out;
  
  &:hover {
    box-shadow: ${shadows.md};
  }
`;