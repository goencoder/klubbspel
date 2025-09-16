// Mapping from Phosphor icons to Iconsax equivalents
// Based on https://iconsax-react.pages.dev/

export const ICON_MAPPING = {
  // Navigation & UI
  ArrowLeft: 'ArrowLeft2',
  Plus: 'Add',
  Check: 'TickCircle',
  
  // Objects & Places
  Buildings: 'Buildings2',
  Trophy: 'Cup',
  Medal: 'Medal',
  Calendar: 'Calendar',
  
  // People & Groups
  UsersThree: 'People',
  Gear: 'Setting2',
  
  // Actions
  MagnifyingGlass: 'SearchNormal1',
  PencilSimple: 'Edit2',
  Trash: 'Trash',
  
  // Charts & Data
  ChartBar: 'Chart',
  
  // Dropdowns & Carets
  CaretDown: 'ArrowDown2',
  CaretUpDown: 'ArrowSwapVertical',
} as const;

export type PhosphorIcon = keyof typeof ICON_MAPPING;
export type IconsaxIcon = typeof ICON_MAPPING[PhosphorIcon];