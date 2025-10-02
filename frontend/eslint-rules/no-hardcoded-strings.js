/**
 * Custom ESLint rule to detect hardcoded strings that should be translated
 * This works in conjunction with the comprehensive Python script for full coverage
 */

// Common technical strings that should be ignored
const TECHNICAL_STRINGS = new Set([
  'use client',
  'GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS',
  'application/json', 'text/plain', 'text/html', 'image/png',
  'Bearer', 'Basic', 'Content-Type', 'Authorization',
  'Accept', 'Accept-Language', 'Origin',
  'NETWORK_ERROR', 'AbortError', 'Request cancelled',
  'success', 'error', 'warning', 'info', 'debug',
  'created', 'updated', 'deleted'
]);

// Patterns for CSS classes and technical identifiers
const TECHNICAL_PATTERNS = [
  /^[a-z][a-z0-9-]*$/,           // kebab-case (CSS classes)
  /^[a-z][a-zA-Z0-9]*$/,         // camelCase (variables)
  /^[a-z][a-z0-9_]*$/,           // snake_case
  /^(text-|bg-|border-|flex|grid|w-|h-|p-|m-|space-|gap-|font-|leading-|tracking-|transform|transition|duration-|ease-|scale-|rotate-|translate-|shadow-|rounded-|overflow-|z-|opacity-|cursor-|select-|pointer-|resize-|outline-|ring-|divide-)/,
  /^(relative|absolute|fixed|sticky|left|right|top|bottom|center|start|end|wrap|nowrap|hidden|block|inline|auto|none)$/,
  /^#[0-9a-fA-F]{3,6}$/,         // hex colors
  /^\d+(px|rem|em|%|vh|vw|deg|ms|s)$/,  // CSS units
  /^data-|^aria-|^className|^class:|^id-|^test-/,  // HTML attributes
];

function isLikelyUserFacingText(text) {
  // Skip empty or very short strings
  if (!text || text.length < 3) return false;
  
  // Skip if it's a known technical string
  if (TECHNICAL_STRINGS.has(text)) return false;
  
  // Skip if it matches technical patterns
  if (TECHNICAL_PATTERNS.some(pattern => pattern.test(text))) return false;
  
  // Skip if it looks like a file path or URL
  if (text.includes('/') && !text.includes(' ')) return false;
  if (text.startsWith('http') || text.startsWith('www') || text.startsWith('mailto:')) return false;
  
  // Skip if it looks like a translation key (contains dots)
  if (text.includes('.') && !text.includes(' ')) return false;
  
  // Likely user-facing if:
  return (
    // Has spaces (likely a sentence)
    text.includes(' ') ||
    // Starts with capital letter and reasonable length
    (text[0] && text[0].toUpperCase() === text[0] && text.length > 4) ||
    // Contains common English words that suggest user-facing content
    /\b(error|failed|success|loading|please|welcome|invalid|required|create|update|delete|save|cancel|confirm|login|logout|sign|email|password|name|user|admin|member|club|player|match|series|tournament|league|score|win|lose|active|inactive)\b/i.test(text) ||
    // Contains punctuation suggesting sentences
    /[.!?:;,]/.test(text) ||
    // Common user-facing patterns
    text.endsWith('...') || text.endsWith(':') || text.endsWith('!') || text.endsWith('?')
  );
}

module.exports = {
  meta: {
    type: 'suggestion',
    docs: {
      description: 'Detect hardcoded strings that should be translated',
      category: 'Best Practices',
    },
    schema: [],
    messages: {
      hardcodedString: 'Hardcoded string "{{text}}" should be translated using t() function',
    },
  },
  
  create(context) {
    return {
      Literal(node) {
        // Only check string literals
        if (typeof node.value !== 'string') return;
        
        const text = node.value;
        
        // Skip strings that are clearly not user-facing
        if (!isLikelyUserFacingText(text)) return;
        
        // Skip if it's already inside a t() function call
        const parent = node.parent;
        if (parent && parent.type === 'CallExpression' && 
            parent.callee && parent.callee.name === 't') {
          return;
        }
        
        // Skip if it's a JSX attribute value for technical attributes
        if (parent && parent.type === 'JSXAttribute') {
          const attrName = parent.name && parent.name.name;
          if (['className', 'style', 'id', 'data-testid', 'aria-label'].includes(attrName)) {
            return;
          }
        }
        
        // Report potential hardcoded string
        context.report({
          node,
          messageId: 'hardcodedString',
          data: {
            text: text.length > 50 ? text.substring(0, 47) + '...' : text
          }
        });
      },
      
      TemplateLiteral(node) {
        // Check template literals for hardcoded strings
        for (const quasi of node.quasis) {
          const text = quasi.value.cooked;
          if (text && isLikelyUserFacingText(text)) {
            context.report({
              node: quasi,
              messageId: 'hardcodedString',
              data: {
                text: text.length > 50 ? text.substring(0, 47) + '...' : text
              }
            });
          }
        }
      }
    };
  }
};