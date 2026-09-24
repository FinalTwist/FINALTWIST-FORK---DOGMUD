## Supported Markdown

### Rules

* Two line breaks starts a new paragraph of text.
* Single line breaks collapse into a single line, UNLESS the previous line ended with a double space (Not my convention!)
* Most wrapping markdown can be nested, bold within emphasis inside of a Heading, etc.

### Headings

Markdown:

`# Heading`

Html:

```
Html tag output: <h1>Heading</h1>
```

Ansitags:

```
<ansi fg="md-h1" bg="md-h1-bg"><ansi fg="md-h1-prefix" bg="md-h1-prefix-bg">.:</ansi> This is a <ansi fg="md-bold" bg="md-bold-bg">HEADING</ansi></ansi>
```

Note: Adding additional #'s will increment the `<h#>`
Note: Ansitags only add the prefix for h1

### Horizontal Lines

Markdown:

```
---
===
:::
```

Html:

```
<hr />
```

Ansitags:

```
<ansi fg="md-hr1" bg="md-hr1-bg">--------------------------------------------------------------------------------</ansi>
<ansi fg="md-hr2" bg="md-hr2-bg">================================================================================</ansi>
<ansi fg="md-hr3" bg="md-hr3-bg">::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::</ansi>
```

### Lists

**Markdown:**

```
- List Item  1
  - List Sub 1
- List Item 2
- List Item 3
```

**Html:**

```
<ul>
  <li>List Item  1
    <ul>
    <li>List Sub 1</li>
    </ul>
  </li>
  <li>List Item 2</li>
  <li>List Item 3</li>
</ul>
```

**Ansitags:**

```
<ansi fg="md-li" bg="md-li-bg">- List Item  1
  <ansi fg="md-li" bg="md-li-bg">- List Sub 1</ansi></ansi>
<ansi fg="md-li" bg="md-li-bg">- List Item 2</ansi>
<ansi fg="md-li" bg="md-li-bg">- List Item 3</ansi>
```

### Emphasis

**Markdown:**

`*Emphasize me*`

**Html:**

```
<em>Emphasize me</em>
```

**Ansitags:**

```
<ansi fg="md-em" bg="md-em-bg">Emphasize me</ansi>
```

### Bold

**Markdown:**

`**Bold me**`

**Html:**

```
<strong>Bold me</strong>
```

**Ansitags:**

```
<ansi fg="md-bold" bg="md-bold-bg">Bold me</ansi>
```

### Special

**Markdown:**

`~I'm Special~`

**Html:**

```
<span data-special="1">I'm Special</span>
```

**Ansitags:**

```
<ansi fg="md-sp1" bg="md-sp1-bg">I'm Special</ansi>
```

**Notes:** Additional wrapping ~'s increment the number: `~~I'm Special~~`, `~~~I'm Special~~~` and so on.
**Notes:** `~~` is typically treated as a strikethrough in markdown.
