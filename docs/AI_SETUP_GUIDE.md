# AI Features Setup Guide

## 🎯 Quick Start

AI features are now **fully implemented** and work out of the box once you configure an AI provider!

### ✅ What's Working Now

- **Natural Language to SQL** - "Describe your query in natural language"
- **Fix Query** - Automatically fix SQL errors
- **Optimize Query** - Get query optimization suggestions
- **Query Explanation** - Understand what a query does
- **6 AI Providers** - OpenAI, Anthropic, Claude Code, Codex, Ollama, HuggingFace

### 🚀 Enabling AI Features

AI is configured through the **Settings → AI** page in the app. No environment variables or restarts required!

#### Quick Start

1. **Launch HowlerOps**
2. **Open Settings** (⚙️ icon or Cmd+,)
3. **Navigate to AI tab**
4. **Configure your preferred provider:**

##### Option 1: OpenAI (Recommended)
- Select "OpenAI" as provider
- Enter your API key: `sk-...`
- Select model: `gpt-4o-mini` (best value) or `gpt-4` (highest quality)
- Click "Save & Test Connection"
- ✅ AI features are now active!

##### Option 2: Anthropic Claude
- Select "Anthropic" as provider
- Enter your API key: `sk-ant-...`
- Select model: `claude-3-5-sonnet-20241022` (recommended)
- Click "Save & Test Connection"
- ✅ AI features are now active!

##### Option 3: Ollama (Local/Free)
- Install Ollama from https://ollama.ai
- Run in terminal: `ollama serve` then `ollama pull sqlcoder:7b`
- In HowlerOps settings:
  - Select "Ollama" as provider
  - Endpoint: `http://localhost:11434` (default)
  - Select model: `sqlcoder:7b` (best for SQL)
  - Click "Save & Test Connection"
- ✅ AI features are now active!

**No restart required!** Changes take effect immediately.

### 📋 All Supported Providers

| Provider | Environment Variable | Default Model | Notes |
|----------|---------------------|---------------|-------|
| **OpenAI** | `OPENAI_API_KEY` | gpt-4o-mini | Best quality, paid |
| **Anthropic** | `ANTHROPIC_API_KEY` | claude-3-5-sonnet | Great for SQL, paid |
| **Ollama** | Auto-detected | sqlcoder:7b | Local, free, private |
| **Codex** | `OPENAI_API_KEY` | code-davinci-002 | Uses OpenAI API |
| **Claude Code** | `CLAUDE_BINARY_PATH` | claude | Requires Claude CLI |
| **HuggingFace** | `HUGGINGFACE_ENDPOINT` | Via Ollama | Local models |

### 🎛️ Advanced Configuration

#### Custom Models

```bash
# OpenAI
export OPENAI_MODEL="gpt-4"
export OPENAI_BASE_URL="https://api.openai.com/v1"

# Anthropic
export ANTHROPIC_MODEL="claude-3-opus-20240229"

# Ollama
export OLLAMA_ENDPOINT="http://localhost:11434"
export OLLAMA_MODEL="codellama:7b"
```

#### Global AI Settings

```bash
# Override default provider
export AI_DEFAULT_PROVIDER="openai"  # or anthropic, ollama, etc.

# Adjust generation parameters
export AI_MAX_TOKENS="2048"
export AI_TEMPERATURE="0.1"
export AI_REQUEST_TIMEOUT="60s"
export AI_RATE_LIMIT_PER_MIN="60"
```

### 🔧 Troubleshooting

#### "AI service not configured" Error

**Cause:** No API keys configured  
**Fix:** Go to Settings → AI and configure a provider with your API key

Alternatively (advanced users only), you can set environment variables before launch:
```bash
export OPENAI_API_KEY="sk-..."
./build/bin/howlerops.app/Contents/MacOS/howlerops
```

#### "Failed to generate SQL" Error

**Possible causes:**
1. **Invalid API key** - Check your key is correct
2. **Network issues** - Check internet connection
3. **Rate limits** - Wait a moment and try again
4. **Model unavailable** - Try a different model

**Fix:**
```bash
# Test your OpenAI connection
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# Or test Anthropic
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01"
```

#### Ollama Not Working

**Fix:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if not running
ollama serve &

# Pull a SQL model
ollama pull sqlcoder:7b

# Restart HowlerOps
```

### 💡 Usage Examples

#### 1. Natural Language to SQL

In the query editor, click "AI Assistant" or use the natural language input:

```
Input: "Show me all users who registered in the last 30 days"

Output:
SELECT * FROM users 
WHERE created_at > NOW() - INTERVAL '30 days'
ORDER BY created_at DESC;

Confidence: 95%
```

#### 2. Fix SQL Errors

When you get a SQL error, click "Fix with AI":

```
Error: column "total_amount" does not exist

Fixed Query:
SELECT SUM(amount) as total_amount FROM orders;

Explanation: Changed to use SUM(amount) aggregation
```

#### 3. Optimize Query

Select a query and click "Optimize":

```
Original:
SELECT * FROM users WHERE email LIKE '%@gmail.com';

Optimized:
SELECT id, name, email FROM users 
WHERE email LIKE '%@gmail.com'
LIMIT 1000;

Impact: Medium-High
Explanation: Added column selection and LIMIT for better performance
```

### 🔐 Security & Privacy

#### API Keys
- Stored only in environment variables (not persisted)
- Never logged or sent anywhere except to the AI provider
- Restart required to change keys

#### Query Privacy
- OpenAI/Anthropic: Queries sent to their APIs
- Ollama: Everything runs locally, nothing sent externally
- No query data is stored by HowlerOps

#### Recommendations
- Use Ollama for sensitive data
- Use OpenAI/Anthropic for best quality
- Don't commit API keys to version control

### 📊 Provider Comparison

| Feature | OpenAI | Anthropic | Ollama |
|---------|--------|-----------|--------|
| **Quality** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Speed** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Cost** | Paid | Paid | **Free** |
| **Privacy** | Cloud | Cloud | **Local** |
| **SQL Expertise** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

**Recommendation:** 
- **Production:** OpenAI (gpt-4o-mini) or Anthropic (claude-3-5-sonnet)
- **Development:** Ollama (sqlcoder:7b) - free and private
- **Sensitive Data:** Ollama only - runs entirely locally

### 🚀 Coming Soon: RAG Features

RAG (Retrieval-Augmented Generation) will auto-enable when AI is enabled, providing:

- **Schema-aware generation** - AI knows your database structure
- **Query history learning** - Learns from your successful queries
- **Context-aware suggestions** - Better autocomplete based on your patterns
- **Business rule integration** - Apply your domain knowledge

No additional configuration needed - it will work automatically with vector embeddings stored locally.

### 🎓 Best Practices

1. **Start with simple prompts**: "Show all users" works better than complex descriptions
2. **Include table names** when possible: "Select from the orders table"
3. **Review generated SQL** before executing (especially with DELETE/UPDATE)
4. **Use Fix Query** for syntax errors - very reliable
5. **Try different providers** - each has strengths

### 📝 Environment Variables Reference

```bash
# Core AI Settings
export AI_DEFAULT_PROVIDER="openai"      # Default: openai
export AI_MAX_TOKENS="2048"              # Default: 2048
export AI_TEMPERATURE="0.1"              # Default: 0.1 (deterministic)
export AI_REQUEST_TIMEOUT="60s"          # Default: 60s
export AI_RATE_LIMIT_PER_MIN="60"        # Default: 60

# OpenAI
export OPENAI_API_KEY="sk-..."           # Required for OpenAI
export OPENAI_BASE_URL="..."             # Optional, default: official API
export OPENAI_ORG_ID="org-..."           # Optional
export OPENAI_MODEL="gpt-4o-mini"        # Optional

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."    # Required for Anthropic
export ANTHROPIC_BASE_URL="..."          # Optional
export ANTHROPIC_VERSION="2023-06-01"    # Optional
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"  # Optional

# Ollama
export OLLAMA_ENDPOINT="http://localhost:11434"  # Default: localhost:11434
export OLLAMA_AUTO_PULL="true"                   # Auto-download models

# HuggingFace (via Ollama)
export HUGGINGFACE_ENDPOINT="http://localhost:11434"
export HUGGINGFACE_AUTO_PULL="true"
```

### 🆘 Support

If AI features aren't working:

1. **Check logs** - Look for "AI service initialized successfully" on startup
2. **Verify provider status** - Check settings/AI panel for provider status
3. **Test API key** - Use curl commands above to verify key works
4. **Try different provider** - Ollama requires no API key
5. **Check GitHub issues** - Others may have solved your problem

### ✅ Success Indicators

When AI is working properly, you should see:

```
# In application logs:
INFO AI service initialized successfully
INFO OpenAI provider enabled
INFO Ollama provider available (default: http://localhost:11434)

# In the UI:
✓ AI Assistant button enabled in query editor
✓ "Describe your query" input field active
✓ Fix Query button appears on SQL errors
✓ Provider status shows "Available" in settings
```

## 🎉 You're All Set!

Your AI features are now fully configured and ready to use. Start by describing a simple query in natural language and watch the magic happen!

