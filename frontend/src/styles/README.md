# Styled Components Architecture Guide

## Overview

Klubbspel frontend uses a hierarchical styled-components system that provides semantic component names, centralized design tokens, and consistent styling patterns while preserving existing Tailwind CSS utilities.

## Architecture

### File Structure
```
src/
├── Styles.js                    # Root tokens and global primitives
├── components/
│   ├── Styles.js                # Shared components across multiple features
│   └── *.tsx                    # Feature components using shared patterns
├── pages/
│   ├── Styles.js                # Page-specific styled components
│   └── *.tsx                    # Page components with consistent layouts
└── styles/
    ├── theme.css                # CSS custom properties (design tokens)
    └── README.md                # This file
```

### Import Hierarchy

1. **Root Tokens** (`src/Styles.js`): Base design system
2. **Shared Components** (`src/components/Styles.js`): Cross-feature patterns  
3. **Page Components** (`src/pages/Styles.js`): Page-specific layouts

```js
// Import pattern from leaf to root
import { SharedEmptyState } from '../components/Styles'
import { colors, spacing } from '../Styles'
```

## Design Token System

### Root Tokens (`src/Styles.js`)

All tokens are mapped to existing CSS custom properties for seamless integration:

```js
export const colors = {
  brand: {
    primary: 'var(--color-primary)',
    primaryHover: 'var(--color-primary/90)',
    accent: 'var(--color-accent-9)'
  },
  text: {
    primary: 'var(--color-foreground)',
    secondary: 'var(--color-muted-foreground)',
    inverse: 'var(--color-primary-foreground)'
  },
  ui: {
    bg: 'var(--color-background)',
    bgSubtle: 'var(--color-muted)',
    border: 'var(--color-border)',
    focus: 'var(--color-ring)',
    success: 'var(--color-chart-1)',
    warning: 'var(--color-chart-4)', 
    danger: 'var(--color-destructive)'
  }
}
```

### Spacing Scale
```js
export const spacing = {
  xs: 'var(--size-1)',    // 4px
  sm: 'var(--size-2)',    // 8px
  md: 'var(--size-3)',    // 12px
  lg: 'var(--size-4)',    // 16px
  xl: 'var(--size-6)',    // 24px
  '2xl': 'var(--size-8)'  // 32px
}
```

## Component Categories

### 1. Shared Components (`src/components/Styles.js`)

Used across multiple features and pages:

#### SharedEmptyState
Standard no-data state presentation
```jsx
<SharedEmptyState>
  <IconComponent size={48} />
  <h3>No items found</h3>
  <p>Try adjusting your search criteria</p>
</SharedEmptyState>
```

#### Status Components
Semantic color-coded components
```jsx
<SuccessIcon size={16} />  // Uses colors.ui.success
<DangerIcon size={16} />   // Uses colors.ui.danger
```

#### Form Components
```jsx
<FormSection>
  <div className="form-group">
    <label>Field Label</label>
    <input type="text" />
  </div>
</FormSection>
```

### 2. Page Components (`src/pages/Styles.js`)

Page-level layout patterns:

#### Page Layout
```jsx
<PageWrapper>
  <PageHeaderSection>
    <HeaderContent>
      <h1>Page Title</h1>
      <p>Page description</p>
    </HeaderContent>
    {/* Action buttons */}
  </PageHeaderSection>
  
  <SearchSection>
    <SearchIcon />
    <Input placeholder="Search..." />
  </SearchSection>
  
  <ContentGrid>
    {/* Cards or content items */}
  </ContentGrid>
</PageWrapper>
```

## Usage Guidelines

### When to Use Styled Components

✅ **Use styled components for:**
- Repeated patterns across multiple components
- Semantic component naming (e.g., `PageHeader` vs `div`)
- Color/spacing that should use design tokens
- Complex layouts that benefit from component composition

❌ **Keep Tailwind classes for:**
- One-off utility styling
- GitHub Spark UI component customization
- Simple responsive utilities
- Positioning and layout helpers

### Naming Conventions

- **Semantic names**: `PageHeader`, `EmptyState`, `SuccessIcon`
- **Descriptive suffixes**: `Container`, `Wrapper`, `Section`, `Group`
- **Avoid generic names**: Use `ActionGroup` not `ButtonContainer`

### Color Usage

Always use semantic tokens, not raw values:

```js
// ✅ Good
color: ${colors.ui.success}
background: ${colors.brand.primary}

// ❌ Avoid
color: #10B981
background: var(--color-green-600)
```

## Adding New Components

### 1. Determine Appropriate Level

- **Root level**: Global primitives used everywhere
- **Shared level**: Components used by 2+ features
- **Page level**: Specific to one page/feature

### 2. Follow Token Pattern

```js
import styled from 'styled-components'
import { colors, spacing, radii } from '../Styles'

export const NewComponent = styled.div`
  background: ${colors.ui.bg};
  padding: ${spacing.lg};
  border-radius: ${radii.md};
`
```

### 3. Export and Document

Add to appropriate Styles.js file and update this README.

## Migration Strategy

For existing components:

1. **Identify patterns**: Look for repeated className combinations
2. **Extract to styled component**: Create semantic component name
3. **Use tokens**: Replace hardcoded values with design tokens
4. **Test thoroughly**: Ensure no visual regressions
5. **Update imports**: Use new component across all instances

## Integration with Existing Systems

### CSS Custom Properties
All tokens map to existing CSS custom properties in `src/styles/theme.css`, ensuring compatibility with the current theming system.

### Tailwind CSS
Styled components complement Tailwind utilities. Use both together:

```jsx
<PageWrapper className="min-h-screen">  {/* Tailwind utility */}
  <HeaderContent>                       {/* Styled component */}
    <h1 className="sr-only">            {/* Tailwind utility */}
      Page Title
    </h1>
  </HeaderContent>
</PageWrapper>
```

### GitHub Spark UI
Preserve GitHub Spark component structure while wrapping with styled components for layout:

```jsx
<Card className="hover:shadow-md">  {/* Spark component */}
  <CardHeader>
    <ActionGroup>                 {/* Styled component */}
      <Button variant="ghost">    {/* Spark component */}
        Edit
      </Button>
    </ActionGroup>
  </CardHeader>
</Card>
```

## Troubleshooting

### Build Issues
- Ensure styled-components is installed: `npm install styled-components`
- Check import paths are correct (relative to file location)
- Verify token imports from root Styles.js

### Theme Issues
- All color tokens use CSS custom properties
- Check theme.css for token definitions
- Verify color values in browser dev tools

### Component Issues
- Check component is exported from appropriate Styles.js
- Ensure semantic HTML structure is maintained
- Test responsive behavior across screen sizes

## Examples

See implementation examples in:
- `src/pages/ClubsPage.tsx` - Page layout patterns
- `src/components/MatchesList.tsx` - Shared component usage
- `src/pages/PlayersPage.tsx` - Consistent pattern application

---

*This styling system was implemented to provide semantic component names, centralized design tokens, and maintainable patterns while preserving the existing Tailwind CSS utility approach.*