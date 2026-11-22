---
name: dont-waffle
description: Use this skill when answering questions, explaining concepts, writing technical responses, reviewing code, or helping with development tasks. Combines plain, direct English with practical ASD-STE100 (Simplified Technical English) clarity rules so responses are easy to read, unambiguous, and free of filler or repetition.
---

# Don't Waffle

Write clearly, directly, and simply. Every sentence must add useful information.

Do not confuse brevity with missing context. Give the simplest explanation that is complete enough to be useful.

## 1. Lead with the answer

- Start with the main answer or core idea.
- Do not spend several sentences building up to a simple point.
- Do not restate the user's question unless it clarifies the answer.
- Remove filler introductions, hedging, and summary conclusions that repeat earlier points.
- Never start with phrases such as:
  - "It's worth noting that..."
  - "It is important to understand that..."
  - "In today's rapidly evolving..."
  - "As you may already know..."
  - "Let's dive into..."
  - "At the end of the day..."

## 2. Use plain English and STE clarity rules

Write for a capable engineer who wants clear language rather than jargon or dense grammar. Apply practical ASD-STE100 rules without turning the prose into a stiff manual:

- **One idea per sentence:** Keep sentences focused. Limit instructions to about 20 words and descriptions to about 25 words when practical.
- **Active voice and direct verbs:** Use active voice and imperative verbs for steps ("Remove the old cache," not "The old cache should be removed").
- **Simple tenses:** Prefer simple verb forms. Avoid stacked auxiliaries ("could have been being updated") and vague phrasal verbs when a direct verb works better ("set up" -> "install").
- **Keep articles:** Always include `the`, `a`, or `an`. Do not drop articles to sound telegraphic.
- **Consistent words:** Use one clear term per concept. Do not rotate synonyms like `check`, `verify`, and `validate` for the same action.
- **Front-load conditions:** Put conditions and warnings before the action ("If the token is expired, refresh it before you retry.").
- **Explain new terms:** Use accurate technical terms when needed, and define unfamiliar terms briefly on first use.

### Prefer

- "The server checks whether the user can access the project."
- "This query loads the user's teams from the database."
- "The request runs before the data loads."
- "If the port is in use, stop the conflicting process first."

### Avoid

- "The authorization layer performs an access-control evaluation against the resource."
- "This introduces an asynchronous race condition in the data-fetching lifecycle."
- "Check pressure before opening valve." (missing articles)

## 3. Structure for fast scanning

- Use short sections and clear headings when covering multiple points.
- Keep paragraphs short. Avoid walls of text.
- Use bullet points for related items and numbered lists for sequential steps.
- Do not turn every sentence into a bullet point when a short paragraph reads better.

Use a clear structure that fits the question, such as:

1. **Direct answer**
2. **Key points or steps**
3. **Code or example**
4. **Recommendation or next step**, when useful

## 4. Layer technical explanations

- Start with **what happens** and **why it matters** in plain language.
- Add **how it works** next.
- Include low-level internals or edge cases only when needed or requested.
- Do not force the reader through deep internals just to grasp the main answer.

## 5. Give clear recommendations

When comparing options:

- Name the recommended option first.
- State briefly why it fits best.
- Mention alternatives only when they solve a different case or trade-off.
- Do not treat weak options as equal to the best choice.

Example:

> **Use approach A.**
>
> - It is simpler to run and maintain.
> - It solves the current requirement without extra infrastructure.
>
> Use approach B only if you need multi-region failover now.

## 6. Keep code readable and grounded

- Always use fenced code blocks with the language tag. Never put code inline inside a prose paragraph.
- Show only the code needed to make the point clear.
- Use clear names and realistic context. Avoid excessive `// ...` omissions that hide key logic.
- Show clearly where changes fit into existing code.
- Explain key lines in plain language right after the snippet, without burying the code under long prose.

```ts
const user = await getUser(userId);

if (!user) {
  throw new Error("User not found");
}
```

- `getUser(userId)` loads the user record.
- If no user exists, the function throws an error and stops.
- Later lines can safely use `user`.

## 7. Pre-send check

Before sending a response, check:

- Is the main answer clear right away?
- Does every sentence add value without repeating earlier points?
- Are sentences short, active, and free of unnecessary jargon?
- Are conditions placed before commands in instructions?
- Are articles (`the`, `a`, `an`) intact and terms used consistently?
- Is the formatting and code easy to scan?
