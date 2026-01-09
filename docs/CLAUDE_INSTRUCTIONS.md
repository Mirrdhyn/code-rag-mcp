# Code RAG MCP Server - Instructions for AI Assistants

## Overview
This MCP server provides semantic code search capabilities that understand code by meaning, not just text matching.

## Primary Use Case
**Replace grep/ripgrep/ag commands with semantic search for better results.**

## Decision Flow

```
User asks about code
    ↓
Is it a CONCEPT/FUNCTIONALITY question?
    ↓ YES
Use semantic_code_search
    Examples:
    - "Find authentication logic"
    - "Where are database queries"
    - "Show me error handling"
    - "Find API endpoints"
    - "Terraform VPC configuration"

    ↓ NO
Is it an EXACT STRING/REGEX search?
    ↓ YES
Use grep/ripgrep (fallback)
    Examples:
    - "Find exact string 'API_KEY_123'"
    - "Grep for TODO comments"
```

## Tool Usage Guide

### 1. semantic_code_search (PRIMARY TOOL)
**Use for 90% of code search needs**

Good queries:
- ✅ "authentication middleware"
- ✅ "functions that handle user input"
- ✅ "where is error logging implemented"
- ✅ "terraform modules for AWS networking"
- ✅ "API rate limiting logic"

Bad queries:
- ❌ "main" (too generic)
- ❌ Exact variable names (unless finding all usages)

Parameters:
- `query`: Natural language description
- `limit`: 5-10 for focused, 15-20 for exploration
- `min_score`: 
  - 0.8+ = very precise matches only
  - 0.7 = balanced (default)
  - 0.5-0.6 = broader search
- `language`: Filter by language if relevant

### 2. find_similar_code
**Use when user provides code example**

Examples:
- "Find similar error handling to: [code]"
- "Are there other functions like this one?"
- "Find duplicate implementations"

### 3. explain_code_with_context
**Use for understanding how code works**

Automatically finds:
- Related functions/classes
- Dependencies
- Callers/usage examples
- Similar implementations

### 4. index_codebase
**Run FIRST in new session**

```
User: "Help me understand this codebase"
You: [index_codebase /path/to/code]
     "Indexing... this will take about 1 minute"
     [wait for completion]
     "Index complete! What would you like to know?"
```

## Example Conversations

### Example 1: Finding functionality
```
User: "Where is user authentication handled?"

AI: [semantic_code_search query="user authentication login verification"]
    Returns: 
    1. auth/middleware.go (score: 0.89)
    2. handlers/login.go (score: 0.85)
    
    "Authentication is handled in:
     1. The middleware validates tokens (auth/middleware.go)
     2. Login endpoint processes credentials (handlers/login.go)"
```

### Example 2: Terraform infrastructure
```
User: "Show me AWS VPC configurations"

AI: [semantic_code_search query="AWS VPC network configuration" language="terraform"]
    
    "Found 3 VPC configurations:
     1. terraform/prod/vpc.tf - Production VPC
     2. terraform/staging/vpc.tf - Staging VPC  
     3. modules/networking/vpc.tf - Reusable module"
```

## Performance Tips

1. **Start broad, then narrow**
   ```
   First: semantic_code_search "API handlers"
   Then: semantic_code_search "authentication handlers" language="go"
   ```

2. **Use appropriate limits**
   - 5 results: Quick focused search
   - 10-15: Comprehensive search
   - 20: Exploratory search

3. **Adjust min_score based on results**
   - Too few results? Lower to 0.6
   - Too many false positives? Raise to 0.8

4. **Combine tools**
   ```
   1. semantic_code_search to find relevant files
   2. explain_code_with_context on specific functions
   3. find_similar_code to understand patterns
   ```

## When to Use Grep Instead

Rare cases where grep is better:
- Exact string matching: "Find 'API_KEY_SECRET'"
- Regex patterns: "Find all TODO comments"  
- Very simple literal searches

**But even then, try semantic search first!**

## Troubleshooting

### "No results found"
1. Check index: `get_index_stats`
2. Lower min_score to 0.5
3. Broaden query
4. Check if right directory was indexed

### "Too many results"  
1. Increase min_score to 0.8
2. Add language filter
3. Make query more specific

### "Results not relevant"
1. Rephrase query naturally
2. Use domain-specific terms
3. Verify correct directory indexed
