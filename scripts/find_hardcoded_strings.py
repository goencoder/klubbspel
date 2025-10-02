#!/usr/bin/env python3
"""
Comprehensive hardcoded string detector for React/TypeScript projects.
This script systematically finds English text that should be translated.

Usage:
    python3 scripts/find_hardcoded_strings.py

This tool helps maintain translation completeness by:
- Automatically scanning all TypeScript/TSX files for hardcoded strings
- Intelligently filtering technical vs user-facing text
- Prioritizing strings by likelihood of being user-facing
- Cross-referencing existing translation keys to find gaps
- Providing file-by-file analysis with line numbers
- Enabling automated maintenance of translation completeness
"""

import os
import re
import json
from typing import Set, List, Tuple

def load_translation_keys(file_path: str) -> Set[str]:
    """Load all translation keys from a translation file."""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        return flatten_keys(data)
    except Exception as e:
        print(f"Error loading {file_path}: {e}")
        return set()

def flatten_keys(obj, prefix=''):
    """Flatten nested JSON object to dot notation keys."""
    keys = set()
    for key, value in obj.items():
        new_key = f"{prefix}.{key}" if prefix else key
        if isinstance(value, dict):
            keys.update(flatten_keys(value, new_key))
        else:
            keys.add(new_key)
    return keys

def extract_translation_calls(content: str) -> Set[str]:
    """Extract t() function calls from TypeScript/TSX content."""
    # Pattern to match t('key') or t("key") calls
    pattern = r't\([\'"]([^\'"]+)[\'"]\)'
    matches = re.findall(pattern, content)
    return set(matches)

def is_likely_user_facing_text(text: str) -> bool:
    """Determine if a string is likely user-facing text that needs translation."""
    
    # Skip if it's obviously technical
    if (
        # CSS classes, attributes, and technical strings
        text.startswith(('data-', 'aria-', 'className', 'class:', 'id-', 'test-', 'btn-', 
                        'text-', 'bg-', 'border-', 'flex', 'grid', 'w-', 'h-', 'p-', 'm-', 
                        'space-', 'gap-', 'font-', 'leading-', 'tracking-', 'transform', 
                        'transition', 'duration-', 'ease-', 'scale-', 'rotate-', 'translate-',
                        'shadow-', 'rounded-', 'overflow-', 'z-', 'opacity-', 'cursor-',
                        'select-', 'pointer-', 'resize-', 'outline-', 'ring-', 'divide-')) or
        # File paths and URLs
        ('/' in text and not ' ' in text) or
        text.startswith(('http', 'www', 'mailto:', 'tel:', 'ftp:', './')) or
        # File extensions and technical patterns
        ('.' in text and len(text.split('.')) == 2 and 
         text.split('.')[1] in ['tsx', 'ts', 'js', 'css', 'png', 'jpg', 'svg', 'json', 
                                'html', 'xml', 'md', 'txt', 'pdf', 'doc', 'xlsx']) or
        # Email patterns
        '@' in text and '.' in text or
        # Short technical strings
        len(text) < 3 or
        # Common technical terms
        text.lower() in ['div', 'span', 'button', 'input', 'form', 'img', 'svg', 'path', 
                        'g', 'rect', 'circle', 'true', 'false', 'null', 'undefined', 'px',
                        'rem', 'em', 'vh', 'vw', 'auto', 'none', 'block', 'inline', 'flex',
                        'grid', 'absolute', 'relative', 'fixed', 'sticky', 'left', 'right',
                        'top', 'bottom', 'center', 'start', 'end', 'wrap', 'nowrap'] or
        # CSS values and hex colors
        text.endswith(('px', 'rem', 'em', '%', 'vh', 'vw', 'deg', 'ms', 's')) or
        re.match(r'^#[0-9a-fA-F]{3,6}$', text) or
        # Numbers and single characters
        text.isdigit() or len(text) == 1 or
        # Empty or whitespace
        text.strip() == '' or
        # Variable names (camelCase, snake_case, kebab-case)
        re.match(r'^[a-z][a-zA-Z0-9]*$', text) or  # camelCase
        re.match(r'^[a-z][a-z0-9_]*$', text) or   # snake_case
        re.match(r'^[a-z][a-z0-9-]*$', text) or   # kebab-case
        # Common non-UI strings
        text in ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS',
                'application/json', 'text/plain', 'text/html', 'image/png',
                'Bearer', 'Basic', 'Content-Type', 'Authorization',
                'success', 'error', 'warning', 'info', 'debug',
                'created', 'updated', 'deleted']
    ):
        return False
    
    # Likely user-facing if:
    return (
        # Has spaces (likely a sentence)
        ' ' in text or
        # Starts with capital letter and reasonable length
        (text[0].isupper() and len(text) > 4) or
        # Contains common English words
        any(word in text.lower() for word in [
            'the', 'and', 'or', 'to', 'for', 'with', 'you', 'your', 'this', 'that',
            'are', 'is', 'was', 'will', 'can', 'have', 'has', 'please', 'error',
            'success', 'failed', 'loading', 'create', 'update', 'delete', 'save',
            'cancel', 'submit', 'confirm', 'welcome', 'login', 'logout', 'sign',
            'email', 'password', 'name', 'user', 'admin', 'member', 'club', 'player',
            'match', 'series', 'tournament', 'league', 'score', 'win', 'lose',
            'active', 'inactive', 'required', 'optional', 'invalid', 'valid'
        ]) or
        # Contains punctuation suggesting sentences
        any(char in text for char in ['.', '!', '?', ':', ';', ',']) or
        # Common user-facing patterns
        text.endswith(('...', ':', '!', '?')) or
        # Button/label text patterns
        (len(text) > 3 and len(text) < 50 and text[0].isupper())
    )

def extract_hardcoded_strings(content: str, file_path: str) -> List[Tuple[str, int]]:
    """Extract potential hardcoded strings from file content."""
    
    hardcoded_strings = []
    lines = content.split('\n')
    
    for line_num, line in enumerate(lines, 1):
        # Skip import statements and comments
        if (line.strip().startswith(('import ', 'export ', '//', '/*', '*/', '<!--')) or
            'eslint-disable' in line):
            continue
            
        # Find all quoted strings in the line
        # Match both single and double quotes, handling escaped quotes
        quote_patterns = [
            r"'([^'\\]*(\\.[^'\\]*)*)'",  # Single quotes with escape handling
            r'"([^"\\]*(\\.[^"\\]*)*)"',  # Double quotes with escape handling
        ]
        
        for pattern in quote_patterns:
            matches = re.finditer(pattern, line)
            for match in matches:
                text = match.group(1)
                # Unescape common escape sequences
                text = text.replace('\\"', '"').replace("\\'", "'").replace('\\n', '\n').replace('\\t', '\t')
                
                if is_likely_user_facing_text(text):
                    hardcoded_strings.append((text, line_num))
    
    return hardcoded_strings

def analyze_frontend_strings(base_path: str = None):
    """Analyze all frontend files for hardcoded strings."""
    
    if base_path is None:
        # Default to current directory structure
        base_path = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    
    frontend_path = os.path.join(base_path, 'frontend', 'src')
    
    if not os.path.exists(frontend_path):
        print(f"Error: Frontend path not found: {frontend_path}")
        return
    
    # Load existing translation keys
    sv_file = os.path.join(base_path, 'frontend', 'src', 'i18n', 'locales', 'sv.json')
    en_file = os.path.join(base_path, 'frontend', 'src', 'i18n', 'locales', 'en.json')
    
    sv_keys = load_translation_keys(sv_file)
    en_keys = load_translation_keys(en_file)
    
    print(f"Loaded {len(sv_keys)} Swedish keys and {len(en_keys)} English keys\n")
    
    all_files = []
    total_hardcoded = []
    used_keys = set()
    
    # Walk through all TypeScript/TSX files
    for root, dirs, files in os.walk(frontend_path):
        # Skip build and dependency directories
        dirs[:] = [d for d in dirs if d not in ['node_modules', 'dist', 'build', '.git']]
        
        for file in files:
            if file.endswith(('.tsx', '.ts')) and not file.endswith('.d.ts'):
                file_path = os.path.join(root, file)
                rel_path = os.path.relpath(file_path, base_path)
                all_files.append(rel_path)
                
                try:
                    with open(file_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                    
                    # Extract translation calls
                    file_keys = extract_translation_calls(content)
                    used_keys.update(file_keys)
                    
                    # Extract hardcoded strings
                    hardcoded = extract_hardcoded_strings(content, rel_path)
                    
                    if hardcoded:
                        print(f"=== {rel_path} ===")
                        for string, line_num in hardcoded:
                            print(f"  Line {line_num}: \"{string}\"")
                            total_hardcoded.append((rel_path, line_num, string))
                        print()
                        
                except Exception as e:
                    print(f"Error processing {file_path}: {e}")
    
    # Summary
    print("=" * 60)
    print("SUMMARY")
    print("=" * 60)
    print(f"Files analyzed: {len(all_files)}")
    print(f"Translation keys used: {len(used_keys)}")
    print(f"Hardcoded strings found: {len(total_hardcoded)}")
    
    # Check for missing translation keys
    missing_sv = used_keys - sv_keys
    missing_en = used_keys - en_keys
    
    if missing_sv or missing_en:
        print(f"\n⚠️  MISSING TRANSLATION KEYS:")
        if missing_sv:
            print(f"Missing in sv.json ({len(missing_sv)}):")
            for key in sorted(missing_sv):
                print(f"  - {key}")
        if missing_en:
            print(f"Missing in en.json ({len(missing_en)}):")
            for key in sorted(missing_en):
                print(f"  - {key}")
    else:
        print("\n✅ All translation keys are properly defined!")
    
    # Priority hardcoded strings (most likely to be user-facing)
    priority_strings = []
    for file_path, line_num, string in total_hardcoded:
        if (
            len(string) > 10 and
            (' ' in string or string.endswith(('!', '?', '.', ':')) or
             any(word in string.lower() for word in [
                 'error', 'success', 'failed', 'loading', 'welcome', 'please',
                 'invalid', 'required', 'create', 'update', 'delete', 'save',
                 'cancel', 'confirm', 'login', 'logout', 'sign', 'email'
             ]))
        ):
            priority_strings.append((file_path, line_num, string))
    
    if priority_strings:
        print(f"\n🔥 HIGH PRIORITY HARDCODED STRINGS ({len(priority_strings)}):")
        print("These are most likely user-facing and should be translated:")
        for file_path, line_num, string in priority_strings:
            print(f"  {file_path}:{line_num} - \"{string}\"")

def main():
    """Main function to run the hardcoded string analysis."""
    print("🔍 Klubbspel Hardcoded String Detection Tool")
    print("=" * 50)
    print("Scanning for hardcoded strings that need translation...\n")
    
    try:
        analyze_frontend_strings()
    except KeyboardInterrupt:
        print("\n\nAnalysis interrupted by user.")
    except Exception as e:
        print(f"\nError during analysis: {e}")

if __name__ == "__main__":
    main()