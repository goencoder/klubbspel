# i18n Hardcoded String Detection Tool

A simple tool to help identify hardcoded strings in the frontend that may need translation.

## Quick Start

```bash
# Check for hardcoded strings (manual inspection)
make i18n.check

# Or run directly 
python3 scripts/validate_i18n.py
```

## What it does

1. **Scans all TypeScript/TSX files** in the frontend
2. **Filters out technical strings** (CSS classes, URLs, file paths, etc.)
3. **Excludes known translation keys** from the translation files
4. **Lists potential hardcoded strings** for manual review

## Sample Output

```
📊 Loaded translation keys:
  sv: 223 keys  
  en: 223 keys

🔑 Translation keys used: 189
🚫 Ignored strings: 241
⚠️  Issues found: 76

❌ HARDCODED STRINGS REQUIRING TRANSLATION:
  frontend/src/components/ClubMergeManager.tsx:45 - "Failed to load merge candidates:"
  frontend/src/pages/LoginPage.tsx:33 - "Welcome! You have been successfully logged in."
```

## Manual Review Process

1. **Run the detection**: `make i18n.check`
2. **Review each flagged string** to determine if it needs translation
3. **For user-facing strings**: Replace with `t('translation.key')` and add to translation files
4. **For technical strings**: Add ignore comments (see below)

## Ignore Comments (Optional)

If the tool flags legitimate technical strings, you can ignore them:

```typescript
// Single line ignore
const error = "NETWORK_ERROR"; // i18n-ignore: Technical constant

// File-level ignore (for utility files)
// i18n-ignore-file: CSS utility classes, not user-facing

// Block ignore
// i18n-ignore-block
const cssClasses = {
  button: "bg-blue-500 hover:bg-blue-700",
  input: "border border-gray-300"
};
// i18n-ignore-block-end
```

## Technical Details

### Automatic Filtering

The tool automatically ignores:
- **CSS classes and styles**: `"flex items-center"`, `"bg-blue-500"`
- **Technical constants**: `"GET"`, `"POST"`, `"application/json"`
- **File paths and URLs**: `"./config.json"`, `"https://api.example.com"`
- **Known translation keys**: Any string found in `sv.json` or `en.json`
- **Short technical strings**: Single characters, numbers, etc.

### Translation Key Detection

Existing translation keys are automatically loaded from:
- `frontend/src/i18n/messages.sv.json`
- `frontend/src/i18n/messages.en.json`

Strings that match these keys are excluded from the report.

## Integration with Development Workflow

- **`make lint`**: No longer includes i18n validation (build won't fail)
- **`make i18n.check`**: Manual inspection tool for developers
- **Manual process**: Review and fix strings as needed during development

This approach gives you visibility into potential translation issues without blocking development workflow.