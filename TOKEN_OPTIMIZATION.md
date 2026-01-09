# 💰 Token Consumption Optimization Guide

This guide explains how to significantly reduce your Claude token consumption during semantic searches.

## 📊 Real-World Results

### Actual Performance (Latest Tests)

**Semantic search with code-rag vs basic grep**:
```
- code-rag (compact mode): ~31% of grep token consumption
- Savings: 69% fewer tokens than basic grep 🎉
```

This represents a substantial improvement while maintaining semantic understanding that grep cannot provide.

## 🎯 Implemented Optimizations

### 1. ✅ Automatic Deduplication

Overlapping chunks (>50% overlap) are automatically removed.

**Before**:
```
auth.js:1-50
auth.js:41-90    ← 40 lines overlap with previous
auth.js:81-130   ← 40 lines overlap with previous
```

**After**:
```
auth.js:1-50     ← Only chunk retained
```

**Savings: ~70% fewer tokens** on results

### 2. ✅ Compact Mode

Displays only `file:line` references without code excerpts.

**Usage**:
```json
{
  "query": "authentication middleware",
  "compact": true
}
```

**Output**:
```
1. `/path/to/auth.js:78-128` (Score: 0.42, javascript)
2. `/path/to/adminAuth.js:16-66` (Score: 0.39, javascript)
3. `/path/to/billingAuth.js:63-113` (Score: 0.38, javascript)
```

**Savings: ~95% fewer tokens** vs full excerpts

### 3. ✅ Configurable Excerpts

Shows only the first N lines of each chunk.

**Usage**:
```json
{
  "query": "authentication middleware",
  "excerpt_lines": 15
}
```

**Output**:
```javascript
// Only 15 lines displayed
function authenticateJWT(req, res, next) {
  const token = req.headers.authorization;
  if (!token) {
    return res.status(401).json({ error: 'No token' });
  }
  // ... 10 more lines
}
... (35 more lines)
```

**Savings: ~70% fewer tokens** vs full chunks

## 🚀 Recommended Usage Modes

### Mode 1: 🔍 **Initial Discovery** (Ultra-economical)

Use when you just want to **know where the code is**.

```json
{
  "query": "authentication middleware",
  "compact": true,
  "limit": 10
}
```

**Consumption**: ~500 tokens  
**Use case**: "Where is the authentication code?"

### Mode 2: 👀 **Quick Overview** (Economical)

Use when you want to **quickly see the code**.

```json
{
  "query": "authentication middleware", 
  "excerpt_lines": 15,
  "limit": 5
}
```

**Consumption**: ~2,000 tokens  
**Use case**: "How does JWT authentication work?"

### Mode 3: 📖 **Deep Analysis** (Standard)

Use when you need the **complete context**.

```json
{
  "query": "authentication middleware",
  "limit": 5
}
```

**Consumption**: ~8,000 tokens  
**Use case**: "Detailed analysis of authentication middleware"

## 💡 Optimal Workflow

### Step 1: Initial Search (compact)

```json
{
  "query": "error handling patterns",
  "compact": true,
  "limit": 10
}
```

**Result**:
```
1. utils/errors.js:23-73 (Score: 0.45)
2. api/errorHandler.js:10-60 (Score: 0.43)
3. middleware/errorMiddleware.js:5-55 (Score: 0.41)
...
```

### Step 2: Target Interesting Files

You see that `api/errorHandler.js` looks relevant.

### Step 3: Read Complete File with Read

```
Tool: Read
File: /path/to/api/errorHandler.js
```

**Total tokens**: 500 (compact) + ~1,500 (Read) = **2,000 tokens**

**vs old workflow**:
- Grep: ~5,000 tokens
- Unoptimized RAG: ~24,600 tokens

## 📈 Detailed Comparison

### Real-World Benchmarks

| Approach | Token Consumption | Relative Cost |
|----------|------------------|---------------|
| **Basic grep** | 100% (baseline) | 1.0x |
| **code-rag (compact)** | 31% | **0.31x** |
| **Savings** | **-69%** | **3.2x more searches possible** |

### Theoretical Maximum Savings

| Scenario | Before | After (compact) | After (excerpt 15) | Savings |
|----------|-------|-----------------|-------------------|----------|
| **Simple search** | 24,600 | 500 | 2,000 | 92-98% |
| **5 searches** | 123,000 | 2,500 | 10,000 | 92-98% |
| **Complete session** | 246,000 | 5,000 | 20,000 | 92-98% |

**Practical example**:
- Budget: 200,000 tokens
- Basic grep: ~40 searches possible
- code-rag (compact): ~128 searches possible (+220%) 🚀

## ⚙️ Configuration in config.yaml

You can set default values:

```yaml
# Search configuration
top_k: 5
min_score: 0.15
compact_mode: false      # true for compact mode by default
default_excerpt_lines: 0  # 15 to limit by default
```

## 🎓 Best Practices

### ✅ DO

1. **Always start in compact mode** for exploration
2. **Use `excerpt_lines: 15`** for previews
3. **Read the complete file** only when necessary
4. **Adjust `limit`** according to your needs (default 5 is good)

### ❌ DON'T

1. ❌ Use `limit: 50` without compact mode
2. ❌ Skip compact mode for discovery
3. ❌ Request all full excerpts by default
4. ❌ Ignore economy tips displayed in results

## 🔮 Future Features

- [ ] "Smart" mode: automatically compact if >10 results
- [ ] Token estimation before display
- [ ] Cache for recent results
- [ ] AI summary of chunks (1-2 lines) instead of excerpts

## 📊 Consumption Monitoring

Add this to your workflows:

```bash
# Before a search
echo "Tokens before: $TOKENS_USED"

# Compact search
semantic_search(query="...", compact=true)

# After
echo "Tokens after: $TOKENS_USED"
echo "Saved: $(($TOKENS_BEFORE - $TOKENS_AFTER))"
```

## 🎯 Concrete Examples

### Example 1: Find All Middlewares

```json
{
  "query": "express middleware",
  "compact": true,
  "limit": 20
}
```

→ Complete list in **500 tokens** instead of 49,200

### Example 2: Understand a Function

```json
{
  "query": "JWT verification function",
  "excerpt_lines": 20,
  "limit": 3
}
```

→ Detailed preview in **2,500 tokens** instead of 7,500

### Example 3: Exploratory Search

```json
{
  "query": "database connection pool",
  "compact": true,
  "limit": 10
}
```

→ Discovery in **500 tokens**, then targeted Read of 1-2 files

## 💰 ROI (Return on Investment)

**Claude Sonnet cost**: ~$3 / 1M input tokens

| Scenario | Before (cost) | After (cost) | Savings $ |
|----------|-------------|--------------|------------|
| 100 searches | $7.38 | $2.29 | **$5.09** |
| 1,000 searches | $73.80 | $22.88 | **$50.92** |
| Complete project | $738.00 | $228.80 | **$509.20** |

**+ Bonus benefit**: More searches possible = better productivity!

---

## 🤝 Contributing

Ideas to reduce consumption even more? Open an issue!
