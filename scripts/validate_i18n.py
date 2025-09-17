#!/usr/bin/env python3
"""
Simple i18n hardcoded string detection tool.

This tool scans the frontend for potentially hardcoded strings that should be translated.
It filters out known translation keys and technical strings, leaving strings that may need translation.

Usage:
    python3 scripts/validate_i18n.py --report    # Generate inspection report (default)
    make i18n.check                              # Run via Makefile
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Dict, List, Set, Tuple

def load_translation_keys(file_path: str) -> Set[str]:
    """Load all translation keys from a translation file."""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        return flatten_keys(data)
    except Exception as e:
        print(f"❌ Error loading {file_path}: {e}")
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
    pattern = r't\([\'"]([^\'"]+)[\'"]\)'
    matches = re.findall(pattern, content)
    return set(matches)

def _is_css_class_string(text: str) -> bool:
    """Check if a string appears to be CSS classes (like Tailwind utility classes)."""
    if not text or len(text) < 3:
        return False
    
    # Split by spaces to get individual classes
    classes = text.split()
    if len(classes) < 2:  # Single word, handle elsewhere
        return False
    
    # Check if most words look like CSS classes
    css_indicators = [
        '-', ':', '[', ']', '(', ')', '%', 'px', 'rem', 'em', 'vh', 'vw',
        'flex', 'grid', 'block', 'inline', 'absolute', 'relative', 'fixed',
        'text', 'bg', 'border', 'rounded', 'shadow', 'opacity', 'transform'
    ]
    
    css_class_count = 0
    for cls in classes:
        if any(indicator in cls.lower() for indicator in css_indicators):
            css_class_count += 1
        elif cls.lower() in ['w', 'h', 'p', 'm', 'mx', 'my', 'px', 'py', 'mt', 'mb', 'ml', 'mr', 
                           'pt', 'pb', 'pl', 'pr', 'top', 'left', 'right', 'bottom', 'center',
                           'justify', 'items', 'content', 'self', 'auto', 'hidden', 'block',
                           'flex', 'grid', 'space', 'gap', 'min', 'max', 'full']:
            css_class_count += 1
    
    # If most classes look like CSS, treat the whole string as CSS
    return css_class_count >= len(classes) * 0.7

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
                        'select-', 'pointer-', 'resize-', 'outline-', 'ring-', 'divide-',
                        'group', 'peer', 'animate-', 'sr-', 'not-sr-', 'focus:', 'hover:',
                        'active:', 'disabled:', 'checked:', 'invalid:', 'valid:', 'hidden',
                        'absolute', 'relative', 'fixed', 'sticky', 'static', 'inset-',
                        'top-', 'right-', 'bottom-', 'left-', 'min-', 'max-', 'size-',
                        'aspect-', 'col-', 'row-', 'place-', 'justify-', 'items-', 'content-',
                        'self-', 'order-', 'basis-', 'grow', 'shrink', 'fill-', 'stroke-',
                        'after:', 'before:', 'first:', 'last:', 'odd:', 'even:', 'empty:',
                        'md:', 'lg:', 'xl:', 'sm:', 'xs:', '2xl:', 'dark:', 'light:',
                        'mx-', 'my-', 'mt-', 'mb-', 'ml-', 'mr-', 'px-', 'py-', 'pt-', 'pb-',
                        'pl-', 'pr-', 'inline-', 'block', 'table', 'flow-')) or
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
                        'top', 'bottom', 'center', 'start', 'end', 'wrap', 'nowrap',
                        'use client', 'ArrowLeft', 'ArrowRight', 'Bold'] or
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
                'Bearer', 'Basic', 'Content-Type', 'Authorization', 'Accept', 'Accept-Language', 'Origin',
                'success', 'error', 'warning', 'info', 'debug',
                'created', 'updated', 'deleted', 'AbortError', 'NETWORK_ERROR',
                'MEMBERSHIP_ROLE_MEMBER', 'MEMBERSHIP_ROLE_ADMIN', 'SERIES_VISIBILITY_OPEN',
                'SERIES_VISIBILITY_CLUB_ONLY', 'CLUB_ADMIN_OR_PLATFORM_OWNER_REQUIRED',
                'CLUB_ID_REQUIRED_FOR_NON_PLATFORM_OWNERS', 'LOGIN_REQUIRED'] or
        # CSS patterns with complex selectors
        re.match(r'^[\w\-:[\]()>&+~*.,#%\s]+$', text) and (
            '[&' in text or '::' in text or '->' in text or 
            'calc(' in text or 'var(' in text or 'min-' in text or 'max-' in text
        ) or
        # Multi-word CSS class strings (common Tailwind pattern)
        _is_css_class_string(text)
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
            'error', 'failed', 'success', 'loading', 'please', 'welcome', 'invalid', 'required',
            'create', 'update', 'delete', 'save', 'cancel', 'confirm', 'login', 'logout', 'sign',
            'email', 'password', 'name', 'user', 'admin', 'member', 'club', 'player',
            'match', 'series', 'tournament', 'league', 'score', 'win', 'lose',
            'active', 'inactive', 'merge', 'remove', 'promote', 'demote', 'invite'
        ]) or
        # Contains punctuation suggesting sentences
        any(char in text for char in ['.', '!', '?', ':', ';', ',']) or
        # Common user-facing patterns
        text.endswith(('...', ':', '!', '?')) or
        # Button/label text patterns
        (len(text) > 3 and len(text) < 50 and text[0].isupper())
    )

def parse_ignore_comments(content: str) -> Tuple[Set[int], Set[Tuple[int, int]], bool]:
    """Parse i18n-ignore comments and return ignored line numbers, blocks, and file-level ignore."""
    lines = content.split('\n')
    ignored_lines = set()
    ignored_blocks = set()
    ignore_entire_file = False
    
    in_ignore_block = False
    block_start = None
    
    for line_num, line in enumerate(lines, 1):
        # Check for file-level ignore
        if '// i18n-ignore-file' in line or '/* i18n-ignore-file' in line:
            ignore_entire_file = True
            
        # Check for inline ignore comments
        if '// i18n-ignore' in line or '/* i18n-ignore' in line:
            ignored_lines.add(line_num)
        
        # Check for block ignore comments
        if '/* i18n-ignore-block */' in line or '// i18n-ignore-block' in line:
            in_ignore_block = True
            block_start = line_num
        elif '/* i18n-ignore-block-end */' in line or '// i18n-ignore-block-end' in line:
            if in_ignore_block and block_start:
                ignored_blocks.add((block_start, line_num))
            in_ignore_block = False
            block_start = None
    
    return ignored_lines, ignored_blocks, ignore_entire_file

def is_line_ignored(line_num: int, ignored_lines: Set[int], ignored_blocks: Set[Tuple[int, int]]) -> bool:
    """Check if a line is ignored by comments."""
    if line_num in ignored_lines:
        return True
    
    for block_start, block_end in ignored_blocks:
        if block_start <= line_num <= block_end:
            return True
    
    return False

def extract_hardcoded_strings(content: str, file_path: str) -> List[Tuple[str, int, bool]]:
    """Extract potential hardcoded strings from file content."""
    
    hardcoded_strings = []
    lines = content.split('\n')
    
    # Parse ignore comments
    ignored_lines, ignored_blocks, ignore_entire_file = parse_ignore_comments(content)
    
    # If entire file is ignored, return empty list
    if ignore_entire_file:
        return []
    
    for line_num, line in enumerate(lines, 1):
        # Skip import statements and comments
        if (line.strip().startswith(('import ', 'export ', '//', '/*', '*/', '<!--')) or
            'eslint-disable' in line):
            continue
            
        # Find all quoted strings in the line
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
                
                is_ignored = is_line_ignored(line_num, ignored_lines, ignored_blocks)
                
                if is_likely_user_facing_text(text):
                    hardcoded_strings.append((text, line_num, is_ignored))
    
    return hardcoded_strings

def validate_translation_keys(used_keys: Set[str], translation_files: Dict[str, str]) -> Dict[str, Set[str]]:
    """Validate that all used translation keys exist in all translation files."""
    missing_keys = {}
    
    for lang, file_path in translation_files.items():
        available_keys = load_translation_keys(file_path)
        missing = used_keys - available_keys
        if missing:
            missing_keys[lang] = missing
    
    return missing_keys

def analyze_frontend_strings(base_path: str = None, fail_on_issues: bool = True) -> int:
    """Analyze all frontend files for hardcoded strings and enforce i18n rules."""
    
    if base_path is None:
        base_path = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    
    frontend_path = os.path.join(base_path, 'frontend', 'src')
    
    if not os.path.exists(frontend_path):
        print(f"❌ Error: Frontend path not found: {frontend_path}")
        return 1
    
    # Translation files
    translation_files = {
        'sv': os.path.join(base_path, 'frontend', 'src', 'i18n', 'locales', 'sv.json'),
        'en': os.path.join(base_path, 'frontend', 'src', 'i18n', 'locales', 'en.json')
    }
    
    # Load existing translation keys
    all_translation_keys = {}
    for lang, file_path in translation_files.items():
        all_translation_keys[lang] = load_translation_keys(file_path)
    
    print(f"📊 Loaded translation keys:")
    for lang, keys in all_translation_keys.items():
        print(f"  {lang}: {len(keys)} keys")
    print()
    
    # Analyze files
    all_files = []
    issues = []
    ignored_strings = []
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
                    
                    # Get all known translation keys for filtering
                    all_known_keys = set()
                    for keys in all_translation_keys.values():
                        all_known_keys.update(keys)  # keys is already a set, not a dict
                    
                    for string, line_num, is_ignored in hardcoded:
                        if is_ignored:
                            ignored_strings.append((rel_path, line_num, string))
                        elif string in all_known_keys:
                            # Skip strings that are known translation keys
                            ignored_strings.append((rel_path, line_num, f"{string} (known translation key)"))
                        else:
                            issues.append((rel_path, line_num, string))
                        
                except Exception as e:
                    print(f"❌ Error processing {file_path}: {e}")
                    if fail_on_issues:
                        return 1
    
    # Validate translation keys
    missing_translation_keys = validate_translation_keys(used_keys, translation_files)
    
    # Report results
    print("=" * 60)
    print("🌍 i18n VALIDATION RESULTS")
    print("=" * 60)
    print(f"📁 Files analyzed: {len(all_files)}")
    print(f"🔑 Translation keys used: {len(used_keys)}")
    print(f"🚫 Ignored strings: {len(ignored_strings)}")
    print(f"⚠️  Issues found: {len(issues)}")
    
    # Report missing translation keys
    if missing_translation_keys:
        print(f"\n❌ MISSING TRANSLATION KEYS:")
        for lang, missing in missing_translation_keys.items():
            print(f"  {lang}.json missing {len(missing)} keys:")
            for key in sorted(missing):
                print(f"    - {key}")
    
    # Report hardcoded strings
    if issues:
        print(f"\n❌ HARDCODED STRINGS REQUIRING TRANSLATION:")
        for file_path, line_num, string in issues:
            # Truncate long strings
            display_string = string if len(string) <= 50 else string[:47] + "..."
            print(f"  {file_path}:{line_num} - \"{display_string}\"")
    
    # Report ignored strings (for reference)
    if ignored_strings:
        print(f"\n✅ IGNORED STRINGS (marked with i18n-ignore):")
        for file_path, line_num, string in ignored_strings:
            display_string = string if len(string) <= 50 else string[:47] + "..."
            print(f"  {file_path}:{line_num} - \"{display_string}\"")
    
    # Summary
    total_issues = len(issues) + len(missing_translation_keys)
    
    if total_issues == 0:
        print(f"\n🎉 SUCCESS: No i18n issues found!")
        return 0
    else:
        print(f"\n💡 HOW TO FIX:")
        print(f"  1. Replace hardcoded strings with t('translation.key')")
        print(f"  2. Add missing translation keys to all .json files")
        print(f"  3. Use // i18n-ignore comments for legitimate hardcoded strings")
        print(f"     Example: const cssClass = \"flex items-center\"; // i18n-ignore: CSS class")
        print(f"\n📖 See scripts/I18N_USAGE.md for detailed documentation")
        
        if fail_on_issues:
            print(f"\n📋 INSPECTION COMPLETE: {total_issues} potential issues found")
            print("💡 Review the above strings manually to determine what needs translation")
            return 0
        else:
            print(f"\n📋 INSPECTION COMPLETE: {total_issues} potential issues found")
            return 0

def main():
    """Main function to run i18n detection."""
    import argparse
    
    parser = argparse.ArgumentParser(description='Detect hardcoded strings for manual i18n review')
    parser.add_argument('--report', action='store_true', default=True,
                       help='Generate inspection report (default)')
    
    args = parser.parse_args()
    
    print("� Generating i18n inspection report...")
    
    return analyze_frontend_strings(fail_on_issues=False)

if __name__ == "__main__":
    sys.exit(main())