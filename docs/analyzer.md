# Analyzer: How It Works

The `analyze` command provides AI-powered analysis of dormant user CSV reports using the GitHub Copilot SDK. This document explains the architecture and data flow.

---

## Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   CSV Report    │────▶│  Stats Aggregator │────▶│  Copilot SDK    │
│ (1000s of rows) │     │  (Go processing)  │     │  (AI Analysis)  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                │                         │
                                ▼                         ▼
                        ┌──────────────────┐     ┌─────────────────┐
                        │ Compact Summary  │     │  AI Response    │
                        │   (~2KB text)    │     │  (Insights)     │
                        └──────────────────┘     └─────────────────┘
```

---

## Key Components

### 1. CSV Statistics Parser (`internal/analysis/stats.go`)

Instead of sending raw CSV data (which can be 30KB+ for large organizations), the analyzer **pre-aggregates statistics** in Go:

| Metric | Description |
|--------|-------------|
| Total Users | Count of all users in the report |
| Active/Dormant | Counts and percentages |
| Email Coverage | How many users have email addresses |
| Activity Distribution | Breakdown by activity type (commits, issues, etc.) |
| Sample Users | Up to 10 active and 10 dormant users as examples |

**Why pre-aggregate?**
- 🚀 **Performance**: ~2KB prompt vs ~34KB raw data
- 💰 **Cost efficiency**: Fewer tokens = lower API costs
- ⚡ **Speed**: Faster AI responses
- 🎯 **Accuracy**: Structured data produces better insights

### 2. Analysis Templates (`internal/analysis/templates.go`)

Five pre-built templates optimize prompts for specific use cases:

| Template | Best For |
|----------|----------|
| **summary** | Executive reporting, quick health checks |
| **trends** | Understanding engagement patterns |
| **risk** | Security reviews, compliance audits |
| **recommendations** | Action planning, offboarding decisions |
| **custom** | Ad-hoc questions about your data |

### 3. Copilot Integration (`internal/analysis/analyzer.go`)

The analyzer uses the [GitHub Copilot SDK for Go](https://github.com/github/copilot-sdk/tree/main/go):

```go
// Create client and session
client := copilot.NewClient(&copilot.ClientOptions{})
session, _ := client.CreateSession(&copilot.SessionConfig{
    Model: "gpt-4o",
})

// Send prompt and stream response
session.Send(copilot.MessageOptions{
    Prompt: formattedPrompt,
})
```

**Safety features:**
- ⏱️ **2-minute timeout** prevents indefinite hangs
- ✅ **Copilot CLI check** before attempting analysis
- 🛡️ **Error handling** for network/API failures

---

## Data Flow

```
1. USER runs: gh dormant-users analyze -f report.csv -t summary

2. COMMAND validates:
   ├── File exists?
   ├── Copilot CLI installed?
   └── Valid template name?

3. PARSER reads CSV and computes:
   ├── User counts (total, active, dormant)
   ├── Activity type breakdown
   ├── Email coverage statistics
   └── Sample user lists (10 each)

4. TEMPLATE formats prompt:
   ┌────────────────────────────────────────────┐
   │ You are analyzing a GitHub organization's  │
   │ user activity report...                    │
   │                                            │
   │ ## Dormant Users Report Statistics         │
   │ - Total Users: 1188                        │
   │ - Active Users: 142 (12.0%)                │
   │ - Dormant Users: 1046 (88.0%)              │
   │ ...                                        │
   └────────────────────────────────────────────┘

5. COPILOT SDK sends to AI model

6. RESPONSE streamed back to terminal
```

---

## Example Output

Running `gh dormant-users analyze -f report.csv -t summary` produces insights like:

> **Executive Summary**
> 
> Your organization shows concerning engagement levels with 88% of users dormant.
> 
> **Key Findings:**
> - Only 142 of 1,188 users showed activity in the review period
> - Issue comments are the most common activity (78% of active users)
> - Direct commits are rare (35% of active users)
> 
> **Recommendations:**
> 1. Review accounts with no activity for potential deactivation
> 2. Investigate why commit activity is low—consider workflow improvements
> 3. Implement quarterly access reviews

---

## Extending the Analyzer

### Adding a New Template

Edit `internal/analysis/templates.go`:

```go
"my-template": {
    Name:        "My Custom Analysis",
    Description: "Description shown in --list-templates",
    Prompt: `Your prompt here with %s placeholder for stats`,
},
```

### Customizing Statistics

Edit `internal/analysis/stats.go` to add new metrics:

```go
type CSVStats struct {
    // ... existing fields
    MyNewMetric int
}
```

Then update `FormatForPrompt()` to include the new data.

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Copilot CLI not available" | Run `gh extension install github/gh-copilot` |
| Timeout errors | Large orgs may need longer timeout; check network |
| Empty response | Verify CSV has correct schema (Username, Active, ActivityTypes) |
| Slow analysis | Use `--prompt-only` to verify prompt size is reasonable |

---

## Security Considerations

- **No raw user data sent to AI**: Only aggregated statistics and anonymized samples
- **Local processing**: CSV parsing happens entirely on your machine
- **Copilot authentication**: Uses your existing GitHub Copilot subscription
- **No data persistence**: Analysis results are not stored by the tool
