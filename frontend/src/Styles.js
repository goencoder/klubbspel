import styled from 'styled-components';

// =============================================================================
// TOKEN SYSTEM - Root Design Tokens
// =============================================================================

// Colors - Mapping to existing CSS custom properties for seamless integration
export const colors = {
  brand: {
    primary: 'var(--color-primary)',
    primaryHover: 'var(--color-primary/90)',
    accent: 'var(--color-accent-9)',
  },
  text: {
    primary: 'var(--color-foreground)',
    secondary: 'var(--color-muted-foreground)',
    inverse: 'var(--color-primary-foreground)',
  },
  ui: {
    bg: 'var(--color-background)',
    bgSubtle: 'var(--color-muted)',
    bgCard: 'var(--color-card)',
    border: 'var(--color-border)',
    focus: 'var(--color-ring)',
    success: 'var(--color-success)',
    warning: 'var(--color-chart-4)',
    danger: 'var(--color-destructive)',
  },
};

// Spacing - Mapping to existing size scale
export const spacing = {
  xs: 'var(--size-1)',    // 4px
  sm: 'var(--size-2)',    // 8px  
  md: 'var(--size-3)',    // 12px
  lg: 'var(--size-4)',    // 16px
  xl: 'var(--size-6)',    // 24px
  '2xl': 'var(--size-8)', // 32px
};

// Border radius - Mapping to existing radius system
export const radii = {
  sm: 'var(--radius-sm)',
  md: 'var(--radius-md)', 
  lg: 'var(--radius-lg)',
  xl: 'var(--radius-xl)',
  pill: 'var(--radius-full)',
};

// Shadows - Common shadow patterns
export const shadows = {
  sm: '0 1px 2px rgba(0,0,0,0.06)',
  md: '0 4px 12px rgba(0,0,0,0.08)',
  lg: '0 8px 24px rgba(0,0,0,0.12)',
};

// Z-index scale
export const z = {
  dropdown: 1000,
  modal: 1100,
  toast: 1200,
};

// =============================================================================
// GLOBAL PRIMITIVES - Common Layout Components
// =============================================================================

// Page container - Consistent max-width and centering
export const PageContainer = styled.div`
  max-width: 1200px;
  margin: 0 auto;
  padding: ${spacing.lg};
  background: ${colors.ui.bg};
`;

// Page header - Common pattern for page titles with actions
export const PageHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: ${spacing.xl};
  
  h1 {
    font-size: 1.875rem;
    font-weight: 700;
    color: ${colors.text.primary};
    margin: 0;
  }
`;

// Section container - Consistent spacing for page sections
export const Section = styled.section`
  margin-bottom: ${spacing['2xl']};
`;

// Card wrapper - Basic card structure (complements Tailwind Card)
export const Card = styled.div`
  background: ${colors.ui.bgCard};
  border: 1px solid ${colors.ui.border};
  border-radius: ${radii.lg};
  box-shadow: ${shadows.sm};
  overflow: hidden;
`;

// Empty state container - Common pattern for no-data states
export const EmptyState = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: ${spacing['2xl']};
  text-align: center;
  
  svg {
    margin-bottom: ${spacing.lg};
    color: ${colors.text.secondary};
  }
  
  h3 {
    font-size: 1.125rem;
    font-weight: 600;
    color: ${colors.text.primary};
    margin-bottom: ${spacing.sm};
  }
  
  p {
    color: ${colors.text.secondary};
    margin: 0;
  }
`;

// =============================================================================
// DEFAULT EXPORT - Token Object for Direct Access
// =============================================================================

export default {
  colors,
  spacing,
  radii,
  shadows,
  z,
};