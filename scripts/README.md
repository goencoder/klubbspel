# Translation Tools

This directory contains tools to help maintain translation completeness in the Klubbspel application.

## find_hardcoded_strings.py

A comprehensive tool to detect hardcoded English strings that should be translated.

### Features

- **Automatic Detection**: Scans all TypeScript/TSX files for hardcoded strings
- **Smart Filtering**: Distinguishes between technical strings and user-facing text
- **Priority Ranking**: Identifies high-priority strings most likely to need translation
- **Translation Key Validation**: Cross-references existing translation keys to find gaps
- **Detailed Analysis**: Provides file-by-file breakdown with line numbers

### Usage

```bash
# Run from the repository root
python3 scripts/find_hardcoded_strings.py

# Or run directly
cd scripts
python3 find_hardcoded_strings.py
```

### How It Works

1. **Scans Frontend Files**: Analyzes all `.tsx` and `.ts` files in `frontend/src/`
2. **Filters Technical Strings**: Excludes CSS classes, URLs, file paths, and other non-user-facing text
3. **Identifies User-Facing Text**: Looks for strings containing spaces, English words, punctuation, or other indicators of user-facing content
4. **Validates Translation Keys**: Compares used translation keys against existing translation files
5. **Prioritizes Results**: Highlights strings most likely to need immediate translation

### Output

The tool provides:
- File-by-file listing of potential hardcoded strings
- Summary statistics
- Missing translation key report
- High-priority strings requiring immediate attention

### Maintenance

Run this tool regularly during development to:
- Catch new hardcoded strings before they reach production
- Ensure translation completeness
- Maintain consistency in internationalization
- Identify gaps in translation coverage

### Integration

This tool can be integrated into CI/CD pipelines to automatically check for translation completeness and prevent hardcoded strings from being merged.